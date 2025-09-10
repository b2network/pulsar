package p2p

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	preexectypes "github.com/b2network/pulsar/pre_execution/types"
)

// OrderBroadcaster handles broadcasting pre-execution orders to peer validators
// This implements the N-txhash-seq synchronization mechanism
type OrderBroadcaster struct {
	mu sync.RWMutex

	// Network configuration
	validatorID   string
	peers         map[string]*PeerConnection // ValidatorID -> Connection
	maxRetries    int
	retryInterval time.Duration

	// Broadcasting state
	sentOrders map[string]*BroadcastRecord // OrderID -> BroadcastRecord

	// Callbacks
	onOrderSent        func(*preexectypes.PreExecutionOrder, []string) // Called when order is broadcast
	onOrderReceived    func(*preexectypes.PreExecutionOrder)           // Called when order is received
	onPeerConnected    func(string)                                    // Called when peer connects
	onPeerDisconnected func(string)                                    // Called when peer disconnects

	// Statistics
	totalSent     uint64
	totalReceived uint64
	failedSends   uint64
	retries       uint64
}

// BroadcastRecord tracks the status of a broadcast operation
type BroadcastRecord struct {
	Order     *preexectypes.PreExecutionOrder `json:"order"`
	Timestamp time.Time                       `json:"timestamp"`
	Attempts  int                             `json:"attempts"`
	Successes []string                        `json:"successes"` // Peers that received successfully
	Failures  []string                        `json:"failures"`  // Peers that failed to receive
	Status    BroadcastStatus                 `json:"status"`
}

// BroadcastStatus represents the status of a broadcast operation
type BroadcastStatus string

const (
	BroadcastPending  BroadcastStatus = "pending"
	BroadcastPartial  BroadcastStatus = "partial"  // Some peers received, some failed
	BroadcastComplete BroadcastStatus = "complete" // All peers received successfully
	BroadcastFailed   BroadcastStatus = "failed"   // All attempts failed
)

// PeerConnection represents a connection to another validator
type PeerConnection struct {
	ValidatorID   string    `json:"validator_id"`
	Address       string    `json:"address"`
	Connected     bool      `json:"connected"`
	LastSeen      time.Time `json:"last_seen"`
	LastSentOrder time.Time `json:"last_sent_order"`

	// Connection statistics
	OrdersSent     uint64 `json:"orders_sent"`
	OrdersReceived uint64 `json:"orders_received"`
	Failures       uint64 `json:"failures"`
}

// OrderMessage represents a network message containing a pre-execution order
type OrderMessage struct {
	Type      string                          `json:"type"` // "pre_execution_order"
	Order     *preexectypes.PreExecutionOrder `json:"order"`
	Sender    string                          `json:"sender"`    // Sender validator ID
	Timestamp time.Time                       `json:"timestamp"` // Message timestamp
	Signature string                          `json:"signature"` // Message signature (simplified)
}

// BroadcasterConfig contains configuration for the order broadcaster
type BroadcasterConfig struct {
	ValidatorID   string        `json:"validator_id"`
	MaxRetries    int           `json:"max_retries"`
	RetryInterval time.Duration `json:"retry_interval"`
	PeerTimeout   time.Duration `json:"peer_timeout"`
}

// NewOrderBroadcaster creates a new pre-execution order broadcaster
func NewOrderBroadcaster(config BroadcasterConfig) *OrderBroadcaster {
	return &OrderBroadcaster{
		validatorID:   config.ValidatorID,
		peers:         make(map[string]*PeerConnection),
		maxRetries:    config.MaxRetries,
		retryInterval: config.RetryInterval,
		sentOrders:    make(map[string]*BroadcastRecord),
	}
}

// AddPeer adds a new peer validator to the broadcaster
func (b *OrderBroadcaster) AddPeer(validatorID, address string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if validatorID == b.validatorID {
		return fmt.Errorf("cannot add self as peer")
	}

	if _, exists := b.peers[validatorID]; exists {
		return fmt.Errorf("peer %s already exists", validatorID)
	}

	// Create new peer connection
	peer := &PeerConnection{
		ValidatorID: validatorID,
		Address:     address,
		Connected:   false,
		LastSeen:    time.Now(),
	}

	b.peers[validatorID] = peer

	// Attempt to connect to the peer
	go b.connectToPeer(peer)

	log.Printf("📡 Added peer: %s (%s)", validatorID, address)
	return nil
}

// RemovePeer removes a peer validator from the broadcaster
func (b *OrderBroadcaster) RemovePeer(validatorID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	peer, exists := b.peers[validatorID]
	if !exists {
		return fmt.Errorf("peer %s not found", validatorID)
	}

	// Disconnect from peer
	b.disconnectFromPeer(peer)

	// Remove from peers map
	delete(b.peers, validatorID)

	log.Printf("📡 Removed peer: %s", validatorID)
	return nil
}

// BroadcastOrder broadcasts a pre-execution order to all connected peers
func (b *OrderBroadcaster) BroadcastOrder(order *preexectypes.PreExecutionOrder) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.peers) == 0 {
		return fmt.Errorf("no peers available for broadcasting")
	}

	// Create broadcast record
	orderID := b.generateOrderID(order)
	record := &BroadcastRecord{
		Order:     order,
		Timestamp: time.Now(),
		Attempts:  0,
		Successes: make([]string, 0),
		Failures:  make([]string, 0),
		Status:    BroadcastPending,
	}

	b.sentOrders[orderID] = record

	// Create order message
	message := &OrderMessage{
		Type:      "pre_execution_order",
		Order:     order,
		Sender:    b.validatorID,
		Timestamp: time.Now(),
		Signature: b.signMessage(order), // Simplified signature
	}

	// Get connected peers
	var connectedPeers []*PeerConnection
	for _, peer := range b.peers {
		if peer.Connected {
			connectedPeers = append(connectedPeers, peer)
		}
	}

	if len(connectedPeers) == 0 {
		record.Status = BroadcastFailed
		return fmt.Errorf("no connected peers available")
	}

	// Broadcast to all connected peers
	go b.broadcastToPeers(message, connectedPeers, record)

	b.totalSent++

	log.Printf("📤 Broadcasting order: %s (seq: %d) to %d peers",
		order.TxHash[:8], order.Sequence, len(connectedPeers))

	return nil
}

// broadcastToPeers sends the order message to specified peers
func (b *OrderBroadcaster) broadcastToPeers(message *OrderMessage, peers []*PeerConnection, record *BroadcastRecord) {
	record.Attempts++

	// Send to each peer concurrently
	var wg sync.WaitGroup
	resultsChan := make(chan BroadcastResult, len(peers))

	for _, peer := range peers {
		wg.Add(1)
		go func(p *PeerConnection) {
			defer wg.Done()

			success := b.sendToPeer(message, p)
			resultsChan <- BroadcastResult{
				PeerID:  p.ValidatorID,
				Success: success,
			}
		}(peer)
	}

	// Wait for all sends to complete
	wg.Wait()
	close(resultsChan)

	// Process results
	b.mu.Lock()
	defer b.mu.Unlock()

	for result := range resultsChan {
		if result.Success {
			record.Successes = append(record.Successes, result.PeerID)
		} else {
			record.Failures = append(record.Failures, result.PeerID)
		}
	}

	// Update broadcast status
	if len(record.Failures) == 0 {
		record.Status = BroadcastComplete
	} else if len(record.Successes) > 0 {
		record.Status = BroadcastPartial
	} else {
		record.Status = BroadcastFailed
		b.failedSends++
	}

	// Retry failed broadcasts if needed
	if len(record.Failures) > 0 && record.Attempts < b.maxRetries {
		go b.retryBroadcast(message, record)
	}

	// Trigger callback
	if b.onOrderSent != nil {
		go b.onOrderSent(message.Order, record.Successes)
	}
}

// sendToPeer sends an order message to a specific peer
func (b *OrderBroadcaster) sendToPeer(message *OrderMessage, peer *PeerConnection) bool {
	// Simulate network send (in production, this would use actual networking)
	// This is where you would implement the actual P2P protocol

	if !peer.Connected {
		return false
	}

	// Serialize message
	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("❌ Failed to serialize message for peer %s: %v", peer.ValidatorID, err)
		return false
	}

	// Simulate network transmission
	success := b.simulateNetworkSend(peer.Address, messageBytes)

	// Update peer statistics
	if success {
		peer.OrdersSent++
		peer.LastSentOrder = time.Now()
	} else {
		peer.Failures++
	}

	return success
}

// retryBroadcast attempts to retry failed broadcasts after a delay
func (b *OrderBroadcaster) retryBroadcast(message *OrderMessage, record *BroadcastRecord) {
	time.Sleep(b.retryInterval)

	b.mu.Lock()
	defer b.mu.Unlock()

	if record.Attempts >= b.maxRetries {
		return
	}

	// Get peers that failed in the last attempt
	var failedPeers []*PeerConnection
	for _, peerID := range record.Failures {
		if peer, exists := b.peers[peerID]; exists && peer.Connected {
			failedPeers = append(failedPeers, peer)
		}
	}

	if len(failedPeers) > 0 {
		log.Printf("🔄 Retrying broadcast for order %s to %d peers (attempt %d/%d)",
			message.Order.TxHash[:8], len(failedPeers), record.Attempts+1, b.maxRetries)

		// Reset failure list for retry
		record.Failures = make([]string, 0)

		// Retry broadcast
		go b.broadcastToPeers(message, failedPeers, record)

		b.retries++
	}
}

// ReceiveOrder processes an incoming pre-execution order from a peer
func (b *OrderBroadcaster) ReceiveOrder(messageBytes []byte, senderAddress string) error {
	// Deserialize message
	var message OrderMessage
	if err := json.Unmarshal(messageBytes, &message); err != nil {
		return fmt.Errorf("failed to deserialize order message: %w", err)
	}

	// Validate message
	if err := b.validateMessage(&message); err != nil {
		return fmt.Errorf("invalid order message: %w", err)
	}

	// Update peer statistics
	b.mu.Lock()
	if peer, exists := b.peers[message.Sender]; exists {
		peer.OrdersReceived++
		peer.LastSeen = time.Now()
	}
	b.totalReceived++
	b.mu.Unlock()

	log.Printf("📥 Received order: %s (seq: %d) from %s",
		message.Order.TxHash[:8], message.Order.Sequence, message.Sender)

	// Trigger callback
	if b.onOrderReceived != nil {
		go b.onOrderReceived(message.Order)
	}

	return nil
}

// Helper methods

// generateOrderID creates a unique ID for tracking broadcast records
func (b *OrderBroadcaster) generateOrderID(order *preexectypes.PreExecutionOrder) string {
	return fmt.Sprintf("%d-%s-%d", order.BaseBlock, order.TxHash, order.Sequence)
}

// signMessage creates a signature for the message (simplified implementation)
func (b *OrderBroadcaster) signMessage(order *preexectypes.PreExecutionOrder) string {
	// In production, this would use proper cryptographic signing
	return fmt.Sprintf("sig_%s_%d", b.validatorID, time.Now().Unix())
}

// validateMessage validates an incoming order message
func (b *OrderBroadcaster) validateMessage(message *OrderMessage) error {
	if message.Type != "pre_execution_order" {
		return fmt.Errorf("invalid message type: %s", message.Type)
	}

	if message.Order == nil {
		return fmt.Errorf("order is nil")
	}

	if message.Sender == "" {
		return fmt.Errorf("sender is empty")
	}

	if message.Sender == b.validatorID {
		return fmt.Errorf("received order from self")
	}

	// Additional validation would go here
	return nil
}

// Network simulation methods (would be replaced with real networking in production)

// connectToPeer simulates connecting to a peer
func (b *OrderBroadcaster) connectToPeer(peer *PeerConnection) {
	// Simulate connection attempt
	time.Sleep(100 * time.Millisecond)

	// For demo purposes, assume connection succeeds
	b.mu.Lock()
	peer.Connected = true
	peer.LastSeen = time.Now()
	b.mu.Unlock()

	log.Printf("🔗 Connected to peer: %s", peer.ValidatorID)

	if b.onPeerConnected != nil {
		go b.onPeerConnected(peer.ValidatorID)
	}
}

// disconnectFromPeer simulates disconnecting from a peer
func (b *OrderBroadcaster) disconnectFromPeer(peer *PeerConnection) {
	peer.Connected = false

	log.Printf("🔌 Disconnected from peer: %s", peer.ValidatorID)

	if b.onPeerDisconnected != nil {
		go b.onPeerDisconnected(peer.ValidatorID)
	}
}

// simulateNetworkSend simulates sending data over the network
func (b *OrderBroadcaster) simulateNetworkSend(address string, data []byte) bool {
	// Simulate network latency
	time.Sleep(10 * time.Millisecond)

	// Simulate 95% success rate
	return time.Now().UnixNano()%100 < 95
}

// Callback setters

func (b *OrderBroadcaster) SetOrderSentCallback(callback func(*preexectypes.PreExecutionOrder, []string)) {
	b.onOrderSent = callback
}

func (b *OrderBroadcaster) SetOrderReceivedCallback(callback func(*preexectypes.PreExecutionOrder)) {
	b.onOrderReceived = callback
}

func (b *OrderBroadcaster) SetPeerConnectedCallback(callback func(string)) {
	b.onPeerConnected = callback
}

func (b *OrderBroadcaster) SetPeerDisconnectedCallback(callback func(string)) {
	b.onPeerDisconnected = callback
}

// Statistics and monitoring

// GetStats returns broadcasting statistics
func (b *OrderBroadcaster) GetStats() BroadcastStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	connectedPeers := 0
	for _, peer := range b.peers {
		if peer.Connected {
			connectedPeers++
		}
	}

	return BroadcastStats{
		TotalPeers:        len(b.peers),
		ConnectedPeers:    connectedPeers,
		TotalSent:         b.totalSent,
		TotalReceived:     b.totalReceived,
		FailedSends:       b.failedSends,
		Retries:           b.retries,
		PendingBroadcasts: b.countPendingBroadcasts(),
	}
}

// countPendingBroadcasts counts broadcasts that are still in progress
func (b *OrderBroadcaster) countPendingBroadcasts() int {
	count := 0
	for _, record := range b.sentOrders {
		if record.Status == BroadcastPending || record.Status == BroadcastPartial {
			count++
		}
	}
	return count
}

// BroadcastResult represents the result of sending to a single peer
type BroadcastResult struct {
	PeerID  string
	Success bool
}

// BroadcastStats contains statistics about broadcasting performance
type BroadcastStats struct {
	TotalPeers        int    `json:"total_peers"`
	ConnectedPeers    int    `json:"connected_peers"`
	TotalSent         uint64 `json:"total_sent"`
	TotalReceived     uint64 `json:"total_received"`
	FailedSends       uint64 `json:"failed_sends"`
	Retries           uint64 `json:"retries"`
	PendingBroadcasts int    `json:"pending_broadcasts"`
}

// GetPeersInfo returns information about all peers
func (b *OrderBroadcaster) GetPeersInfo() []PeerConnection {
	b.mu.RLock()
	defer b.mu.RUnlock()

	peers := make([]PeerConnection, 0, len(b.peers))
	for _, peer := range b.peers {
		peers = append(peers, *peer)
	}

	return peers
}
