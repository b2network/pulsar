package keeper

import (
	"fmt"
	"sync"
	"time"

	"github.com/b2network/pulsar/pre_execution/types"
)

// TxSequencer manages the ordering of pre-executed transactions
// It ensures that transactions are executed and applied in a consistent order across all nodes
type TxSequencer struct {
	mu sync.RWMutex

	// Configuration
	baseHeight  int64         // Current base block height
	maxPending  uint32        // Maximum pending transactions
	syncTimeout time.Duration // Timeout for sequence synchronization

	// Sequence management
	sequences  map[string]uint32 // TxHash -> sequence number mapping
	orderedTxs []string          // Ordered list of transaction hashes
	nextSeq    uint32            // Next available sequence number

	// Synchronization
	pendingOrders map[string]*types.PreExecutionOrder // Orders waiting to be applied
	validators    map[string]bool                     // Known validator set

	// Statistics
	totalOrders    uint64 // Total orders processed
	ordersReceived uint64 // Orders received from other nodes
	ordersSent     uint64 // Orders broadcast to other nodes

	// Callbacks
	onOrderReceived func(*types.PreExecutionOrder) // Called when order is received from peer
	onOrderApplied  func(*types.PreExecutionOrder) // Called when order is applied locally
}

// PreExecutionOrder represents an order message for transaction sequencing
// This is broadcast to other nodes to maintain consistent execution order
type PreExecutionOrder struct {
	// BaseBlock is the reference block height (N)
	BaseBlock int64 `json:"base_block"`

	// TxHash uniquely identifies the transaction
	TxHash string `json:"tx_hash"`

	// Sequence is the order number for this transaction after BaseBlock
	Sequence uint32 `json:"sequence"`

	// Timestamp records when this order was created
	Timestamp time.Time `json:"timestamp"`

	// ValidatorID identifies the validator that created this order
	ValidatorID string `json:"validator_id"`

	// Priority can be used for order resolution in case of conflicts
	Priority uint32 `json:"priority"`
}

// SequencerConfig contains configuration for the transaction sequencer
type SequencerConfig struct {
	MaxPending     uint32        `json:"max_pending"`     // Maximum pending transactions
	SyncTimeout    time.Duration `json:"sync_timeout"`    // Synchronization timeout
	OrderBroadcast bool          `json:"order_broadcast"` // Whether to broadcast orders
}

// NewTxSequencer creates a new transaction sequencer
func NewTxSequencer(baseHeight int64, config SequencerConfig) *TxSequencer {
	return &TxSequencer{
		baseHeight:    baseHeight,
		maxPending:    config.MaxPending,
		syncTimeout:   config.SyncTimeout,
		sequences:     make(map[string]uint32),
		orderedTxs:    make([]string, 0),
		pendingOrders: make(map[string]*types.PreExecutionOrder),
		validators:    make(map[string]bool),
		nextSeq:       1, // Start from 1 (0 reserved)
	}
}

// AddTransaction adds a new transaction to the sequence
// Returns the sequence number assigned to this transaction
func (s *TxSequencer) AddTransaction(txHash string, validatorID string, priority uint32) (uint32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if transaction already exists
	if _, exists := s.sequences[txHash]; exists {
		return s.sequences[txHash], nil
	}

	// Check capacity limits
	if len(s.orderedTxs) >= int(s.maxPending) {
		return 0, fmt.Errorf("sequencer at capacity: %d transactions pending", len(s.orderedTxs))
	}

	// Assign sequence number
	sequence := s.nextSeq
	s.nextSeq++

	// Store sequence mapping
	s.sequences[txHash] = sequence
	s.orderedTxs = append(s.orderedTxs, txHash)

	// Create order message
	order := &types.PreExecutionOrder{
		BaseBlock:   s.baseHeight,
		TxHash:      txHash,
		Sequence:    sequence,
		Timestamp:   time.Now(),
		ValidatorID: validatorID,
		Priority:    priority,
	}

	s.totalOrders++

	// Trigger callback if set
	if s.onOrderApplied != nil {
		go s.onOrderApplied(order)
	}

	return sequence, nil
}

// GetSequence returns the sequence number for a given transaction
func (s *TxSequencer) GetSequence(txHash string) (uint32, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sequence, exists := s.sequences[txHash]
	return sequence, exists
}

// GetOrderedTransactions returns transactions in sequence order
func (s *TxSequencer) GetOrderedTransactions() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	ordered := make([]string, len(s.orderedTxs))
	copy(ordered, s.orderedTxs)
	return ordered
}

// GetTransactionsInRange returns transactions with sequences in the given range
func (s *TxSequencer) GetTransactionsInRange(startSeq, endSeq uint32) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []string
	for _, txHash := range s.orderedTxs {
		if seq, exists := s.sequences[txHash]; exists {
			if seq >= startSeq && seq <= endSeq {
				result = append(result, txHash)
			}
		}
	}

	return result
}

// ReceiveOrder processes an order received from another validator
func (s *TxSequencer) ReceiveOrder(order *types.PreExecutionOrder) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate order
	if err := s.validateOrder(order); err != nil {
		return fmt.Errorf("invalid order received: %w", err)
	}

	// Check if we already have this transaction
	if existingSeq, exists := s.sequences[order.TxHash]; exists {
		if existingSeq != order.Sequence {
			// Sequence conflict - handle based on priority and timestamp
			return s.handleSequenceConflict(order, existingSeq)
		}
		return nil // Already have this order
	}

	// Check if the sequence number is already taken
	if s.isSequenceTaken(order.Sequence) {
		// Store as pending and try to resolve later
		s.pendingOrders[order.TxHash] = order
		return nil
	}

	// Apply the order
	if err := s.applyOrder(order); err != nil {
		return fmt.Errorf("failed to apply order: %w", err)
	}

	s.ordersReceived++

	// Trigger callback if set
	if s.onOrderReceived != nil {
		go s.onOrderReceived(order)
	}

	return nil
}

// validateOrder validates an incoming order message
func (s *TxSequencer) validateOrder(order *types.PreExecutionOrder) error {
	// Check base block height
	if order.BaseBlock != s.baseHeight {
		return fmt.Errorf("order base block %d does not match current base %d",
			order.BaseBlock, s.baseHeight)
	}

	// Check sequence bounds
	if order.Sequence == 0 {
		return fmt.Errorf("invalid sequence number: 0")
	}

	// Check validator
	if order.ValidatorID == "" {
		return fmt.Errorf("missing validator ID")
	}

	// Check timestamp (not too old or too far in future)
	now := time.Now()
	if order.Timestamp.Before(now.Add(-s.syncTimeout)) {
		return fmt.Errorf("order timestamp too old")
	}
	if order.Timestamp.After(now.Add(s.syncTimeout)) {
		return fmt.Errorf("order timestamp too far in future")
	}

	return nil
}

// isSequenceTaken checks if a sequence number is already assigned
func (s *TxSequencer) isSequenceTaken(sequence uint32) bool {
	for _, seq := range s.sequences {
		if seq == sequence {
			return true
		}
	}
	return false
}

// applyOrder applies a validated order to the sequencer state
func (s *TxSequencer) applyOrder(order *types.PreExecutionOrder) error {
	// Add to sequences
	s.sequences[order.TxHash] = order.Sequence

	// Insert in correct position in ordered list
	s.insertOrdered(order.TxHash, order.Sequence)

	// Update next sequence if necessary
	if order.Sequence >= s.nextSeq {
		s.nextSeq = order.Sequence + 1
	}

	return nil
}

// insertOrdered inserts a transaction hash in the correct position based on sequence
func (s *TxSequencer) insertOrdered(txHash string, sequence uint32) {
	// Find insertion point
	insertIndex := len(s.orderedTxs)
	for i, existingTxHash := range s.orderedTxs {
		if existingSeq, exists := s.sequences[existingTxHash]; exists {
			if sequence < existingSeq {
				insertIndex = i
				break
			}
		}
	}

	// Insert at the correct position
	s.orderedTxs = append(s.orderedTxs, "")
	copy(s.orderedTxs[insertIndex+1:], s.orderedTxs[insertIndex:])
	s.orderedTxs[insertIndex] = txHash
}

// handleSequenceConflict resolves conflicts when multiple orders have different sequences for the same tx
func (s *TxSequencer) handleSequenceConflict(newOrder *types.PreExecutionOrder, existingSeq uint32) error {
	// Simple conflict resolution: higher priority wins, then earlier timestamp
	existingTxHash := ""
	for txHash, seq := range s.sequences {
		if seq == existingSeq {
			existingTxHash = txHash
			break
		}
	}

	if existingTxHash == "" {
		// This shouldn't happen, but handle it gracefully
		return s.applyOrder(newOrder)
	}

	// For now, keep the existing order (first-come-first-served)
	// In production, this could use more sophisticated conflict resolution
	return nil
}

// RemoveTransaction removes a transaction from the sequencer (e.g., after block confirmation)
func (s *TxSequencer) RemoveTransaction(txHash string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.sequences[txHash]
	if !exists {
		return false
	}

	// Remove from sequences map
	delete(s.sequences, txHash)

	// Remove from ordered list
	for i, hash := range s.orderedTxs {
		if hash == txHash {
			s.orderedTxs = append(s.orderedTxs[:i], s.orderedTxs[i+1:]...)
			break
		}
	}

	// Remove from pending orders if exists
	delete(s.pendingOrders, txHash)

	return true
}

// UpdateBaseHeight updates the base block height and clears old sequences
func (s *TxSequencer) UpdateBaseHeight(newHeight int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if newHeight <= s.baseHeight {
		return
	}

	s.baseHeight = newHeight

	// Clear all sequences as they're now invalid
	s.sequences = make(map[string]uint32)
	s.orderedTxs = make([]string, 0)
	s.pendingOrders = make(map[string]*types.PreExecutionOrder)
	s.nextSeq = 1
}

// ProcessPendingOrders attempts to apply any pending orders that can now be resolved
func (s *TxSequencer) ProcessPendingOrders() {
	s.mu.Lock()
	defer s.mu.Unlock()

	applied := make([]string, 0)

	for txHash, order := range s.pendingOrders {
		// Check if this order can now be applied
		if !s.isSequenceTaken(order.Sequence) {
			if err := s.applyOrder(order); err == nil {
				applied = append(applied, txHash)
			}
		}
	}

	// Remove successfully applied orders from pending
	for _, txHash := range applied {
		delete(s.pendingOrders, txHash)
	}
}

// GetStats returns sequencer statistics
func (s *TxSequencer) GetStats() SequencerStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return SequencerStats{
		BaseHeight:     s.baseHeight,
		PendingTxs:     len(s.orderedTxs),
		MaxPending:     int(s.maxPending),
		NextSequence:   s.nextSeq,
		TotalOrders:    s.totalOrders,
		OrdersReceived: s.ordersReceived,
		OrdersSent:     s.ordersSent,
		PendingOrders:  len(s.pendingOrders),
	}
}

// SequencerStats contains statistics about the transaction sequencer
type SequencerStats struct {
	BaseHeight     int64  `json:"base_height"`     // Current base block height
	PendingTxs     int    `json:"pending_txs"`     // Number of pending transactions
	MaxPending     int    `json:"max_pending"`     // Maximum pending transactions
	NextSequence   uint32 `json:"next_sequence"`   // Next sequence number to assign
	TotalOrders    uint64 `json:"total_orders"`    // Total orders processed
	OrdersReceived uint64 `json:"orders_received"` // Orders received from peers
	OrdersSent     uint64 `json:"orders_sent"`     // Orders sent to peers
	PendingOrders  int    `json:"pending_orders"`  // Orders waiting to be applied
}

// SetOrderReceivedCallback sets the callback for when an order is received
func (s *TxSequencer) SetOrderReceivedCallback(callback func(*types.PreExecutionOrder)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onOrderReceived = callback
}

// SetOrderAppliedCallback sets the callback for when an order is applied
func (s *TxSequencer) SetOrderAppliedCallback(callback func(*types.PreExecutionOrder)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onOrderApplied = callback
}

// Clear removes all transactions and resets the sequencer state
func (s *TxSequencer) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sequences = make(map[string]uint32)
	s.orderedTxs = make([]string, 0)
	s.pendingOrders = make(map[string]*types.PreExecutionOrder)
	s.nextSeq = 1
}
