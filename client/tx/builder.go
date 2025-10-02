package tx

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/b2network/pulsar/crypto/keys"
	"github.com/b2network/pulsar/crypto/signature"
	"github.com/b2network/pulsar/modules/fee/config"
	"github.com/b2network/pulsar/modules/fee/types"
	commontypes "github.com/b2network/pulsar/types"
)

// TxBuilder handles transaction construction, signing, and broadcasting
type TxBuilder struct {
	chainID      string
	keyring      keys.Keyring
	signer       *signature.Signer
	verifier     *signature.Verifier
	gasStandards config.GasStandards
	feeConfig    *FeeConfig
}

// FeeConfig contains fee calculation configuration
type FeeConfig struct {
	EnabledDenoms     []types.FeeDenom
	GasPriceOracle    types.GasPriceOracle
	DynamicFactors    types.DynamicGasFactors
	FeeDistribution   types.FeeDistributionConfig
	UseIntelligentFee bool
}

// NewTxBuilder creates a new transaction builder
func NewTxBuilder(chainID, keyDir string) (*TxBuilder, error) {
	keyring, err := keys.NewFileKeyring(keyDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create keyring: %w", err)
	}

	signer := signature.NewSigner(chainID, keyring)
	verifier := signature.NewVerifier(chainID)

	// Initialize with default gas standards and fee configuration
	gasStandards := config.DefaultGasStandards()
	feeConfig := &FeeConfig{
		EnabledDenoms:     getDefaultFeeDenoms(),
		DynamicFactors:    getDefaultDynamicFactors(),
		FeeDistribution:   getDefaultFeeDistribution(),
		UseIntelligentFee: true,
	}

	return &TxBuilder{
		chainID:      chainID,
		keyring:      keyring,
		signer:       signer,
		verifier:     verifier,
		gasStandards: gasStandards,
		feeConfig:    feeConfig,
	}, nil
}

// NewTxBuilderWithFeeConfig creates a transaction builder with custom fee configuration
func NewTxBuilderWithFeeConfig(chainID, keyDir string, feeConfig *FeeConfig) (*TxBuilder, error) {
	txBuilder, err := NewTxBuilder(chainID, keyDir)
	if err != nil {
		return nil, err
	}

	if feeConfig != nil {
		txBuilder.feeConfig = feeConfig
	}

	return txBuilder, nil
}

// TxConfig contains transaction configuration
type TxConfig struct {
	From          string // Key name or address
	Fees          string // Fee amount, e.g., "100ubtc"
	Gas           string // Gas limit, "auto" for automatic calculation
	GasPrices     string // Gas prices, e.g., "0.1ubtc"
	Memo          string // Transaction memo
	AccountNum    uint64 // Account number
	Sequence      uint64 // Account sequence
	TimeoutHeight uint64 // Block timeout height

	// Enhanced fee options
	FeeDenom      string      // Preferred fee denomination
	FeePriority   FeePriority // Fee priority: low, normal, high, fast
	AutoFee       bool        // Enable automatic fee calculation
	MaxFee        string      // Maximum fee willing to pay
	FeeMultiplier float64     // Multiplier for calculated fees (default: 1.0)
}

// FeePriority represents transaction fee priority levels
type FeePriority string

const (
	FeePriorityLow    FeePriority = "low"
	FeePriorityNormal FeePriority = "normal"
	FeePriorityHigh   FeePriority = "high"
	FeePriorityFast   FeePriority = "fast"
)

// Tx represents a signed transaction
type Tx struct {
	Body      TxBody                `json:"body"`
	AuthInfo  AuthInfo              `json:"auth_info"`
	Signatures []string             `json:"signatures"`
}

// TxBody contains the transaction body
type TxBody struct {
	Messages      []json.RawMessage `json:"messages"`
	Memo          string            `json:"memo"`
	TimeoutHeight uint64            `json:"timeout_height"`
}

// AuthInfo contains transaction authentication info
type AuthInfo struct {
	SignerInfos []SignerInfo    `json:"signer_infos"`
	Fee         Fee             `json:"fee"`
}

// SignerInfo contains signer information
type SignerInfo struct {
	PublicKey string `json:"public_key"`
	ModeInfo  ModeInfo `json:"mode_info"`
	Sequence  uint64   `json:"sequence"`
}

// ModeInfo contains signature mode info
type ModeInfo struct {
	Single *ModeInfoSingle `json:"single,omitempty"`
}

// ModeInfoSingle contains single signature mode info
type ModeInfoSingle struct {
	Mode string `json:"mode"`
}

// Fee represents transaction fee
type Fee struct {
	Amount   []commontypes.Coin `json:"amount"`
	GasLimit uint64             `json:"gas_limit"`
	Payer    string             `json:"payer"`
	Granter  string             `json:"granter"`
}

// BuildAndSign builds and signs a transaction
func (tb *TxBuilder) BuildAndSign(msgs []commontypes.Msg, config TxConfig, passphrase string) (*Tx, error) {
	// Get signer info
	keyInfo, err := tb.keyring.GetKey(config.From)
	if err != nil {
		return nil, fmt.Errorf("failed to get key %s: %w", config.From, err)
	}

	// Serialize messages
	var rawMsgs []json.RawMessage
	for _, msg := range msgs {
		msgBytes, err := json.Marshal(msg)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal message: %w", err)
		}
		rawMsgs = append(rawMsgs, msgBytes)
	}

	// Parse fee with intelligent calculation
	fee, err := tb.parseEnhancedFee(config, msgs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse fee: %w", err)
	}

	// Build transaction body
	txBody := TxBody{
		Messages:      rawMsgs,
		Memo:          config.Memo,
		TimeoutHeight: config.TimeoutHeight,
	}

	// Build auth info
	authInfo := AuthInfo{
		SignerInfos: []SignerInfo{
			{
				PublicKey: fmt.Sprintf("0x%x", keyInfo.PubKey.Bytes()),
				ModeInfo: ModeInfo{
					Single: &ModeInfoSingle{
						Mode: "SIGN_MODE_DIRECT",
					},
				},
				Sequence: config.Sequence,
			},
		},
		Fee: fee,
	}

	// Create sign document
	signDoc := &signature.SignDoc{
		ChainID:       tb.chainID,
		AccountNumber: config.AccountNum,
		Sequence:      config.Sequence,
		Fee:           signature.Fee{
			Amount: convertCoinsToSignature(fee.Amount),
			Gas:    fee.GasLimit,
		},
		Messages:      convertMsgsToSignature(msgs),
		Memo:          config.Memo,
		TimeoutHeight: config.TimeoutHeight,
	}

	// Sign the transaction
	sig, err := tb.signer.SignTransaction(
		config.From,
		passphrase,
		signDoc.AccountNumber,
		signDoc.Sequence,
		signDoc.Fee,
		signDoc.Messages,
		signDoc.Memo,
		signDoc.TimeoutHeight,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Build final transaction
	tx := &Tx{
		Body:     txBody,
		AuthInfo: authInfo,
		Signatures: []string{
			fmt.Sprintf("0x%x", sig.Signature),
		},
	}

	return tx, nil
}

// VerifyTx verifies a transaction signature
func (tb *TxBuilder) VerifyTx(tx *Tx) error {
	if len(tx.Signatures) == 0 {
		return fmt.Errorf("no signatures found")
	}

	if len(tx.AuthInfo.SignerInfos) == 0 {
		return fmt.Errorf("no signer info found")
	}

	// Parse public key
	pubKeyHex := tx.AuthInfo.SignerInfos[0].PublicKey
	if len(pubKeyHex) > 2 && pubKeyHex[:2] == "0x" {
		pubKeyHex = pubKeyHex[2:]
	}

	pubKeyBytes := make([]byte, len(pubKeyHex)/2)
	for i := 0; i < len(pubKeyHex); i += 2 {
		b, err := strconv.ParseUint(pubKeyHex[i:i+2], 16, 8)
		if err != nil {
			return fmt.Errorf("invalid public key hex: %w", err)
		}
		pubKeyBytes[i/2] = byte(b)
	}

	pubKey, err := keys.NewSecp256k1PubKey(pubKeyBytes)
	if err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}

	// Parse signature
	sigHex := tx.Signatures[0]
	if len(sigHex) > 2 && sigHex[:2] == "0x" {
		sigHex = sigHex[2:]
	}

	sigBytes := make([]byte, len(sigHex)/2)
	for i := 0; i < len(sigHex); i += 2 {
		b, err := strconv.ParseUint(sigHex[i:i+2], 16, 8)
		if err != nil {
			return fmt.Errorf("invalid signature hex: %w", err)
		}
		sigBytes[i/2] = byte(b)
	}

	// Create signature for verification
	sig := &signature.Signature{
		PubKey:    pubKey,
		Signature: sigBytes,
	}

	// Recreate sign document
	signDoc := &signature.SignDoc{
		ChainID:       tb.chainID,
		AccountNumber: 0, // Would need to be passed in
		Sequence:      tx.AuthInfo.SignerInfos[0].Sequence,
		Fee: signature.Fee{
			Amount: convertCoinsToSignature(tx.AuthInfo.Fee.Amount),
			Gas:    tx.AuthInfo.Fee.GasLimit,
		},
		Messages:      []signature.Message{}, // Would need to be reconstructed
		Memo:          tx.Body.Memo,
		TimeoutHeight: tx.Body.TimeoutHeight,
	}

	// Get sign bytes
	signBytes, err := signDoc.GetSignBytes()
	if err != nil {
		return fmt.Errorf("failed to get sign bytes: %w", err)
	}

	// Verify signature
	if !sig.Verify(signBytes) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// EstimateGas estimates gas for a transaction using intelligent calculation
func (tb *TxBuilder) EstimateGas(msgs []commontypes.Msg) (uint64, error) {
	if !tb.feeConfig.UseIntelligentFee {
		// Fallback to simple estimation
		baseGas := uint64(50000)
		msgGas := uint64(10000) * uint64(len(msgs))
		return baseGas + msgGas, nil
	}

	// Use intelligent gas estimation
	totalGas := tb.gasStandards.BaseTxGas

	for _, msg := range msgs {
		msgGas, err := tb.estimateMessageGas(msg)
		if err != nil {
			// Log error and use fallback
			msgGas = 10000 // Fallback gas per message
		}
		totalGas += msgGas
	}

	// Apply dynamic factors
	configFactors := config.DynamicGasFactors{
		SizeMultiplier:   tb.feeConfig.DynamicFactors.SizeMultiplier,
		ComplexityFactor: tb.feeConfig.DynamicFactors.ComplexityFactor,
		NetworkFactor:    tb.feeConfig.DynamicFactors.NetworkFactor,
		StorageFactor:    tb.feeConfig.DynamicFactors.StorageFactor,
	}
	dynamicGas := tb.gasStandards.CalculateDynamicGas(
		"generic", // Module name (would be determined from message type)
		"transaction",
		configFactors,
		map[string]interface{}{
			"message_count": len(msgs),
			"total_gas":     totalGas,
		},
	)

	return dynamicGas, nil
}

// estimateMessageGas estimates gas for a specific message
func (tb *TxBuilder) estimateMessageGas(msg commontypes.Msg) (uint64, error) {
	msgType := fmt.Sprintf("%T", msg)

	// Parse module and message type from full type name
	parts := strings.Split(msgType, ".")
	if len(parts) < 2 {
		return 10000, nil // Default fallback
	}

	moduleName := "bank" // Default, would be parsed from message type
	messageType := parts[len(parts)-1]

	// Use gas standards to get base gas for this message type
	switch moduleName {
	case "bank":
		switch messageType {
		case "MsgSend":
			return tb.gasStandards.BankGasConfig.Send, nil
		case "MsgMultiSend":
			return tb.gasStandards.BankGasConfig.MultiSend, nil
		}
	case "coin":
		switch messageType {
		case "MsgMint":
			return tb.gasStandards.CoinGasConfig.Mint, nil
		case "MsgBurn":
			return tb.gasStandards.CoinGasConfig.Burn, nil
		}
	case "staking":
		switch messageType {
		case "MsgDelegate":
			return tb.gasStandards.StakingGasConfig.Delegate, nil
		case "MsgUndelegate":
			return tb.gasStandards.StakingGasConfig.Undelegate, nil
		}
	}

	// Default gas for unknown message types
	return 15000, nil
}

// BroadcastTx broadcasts a transaction (placeholder)
func (tb *TxBuilder) BroadcastTx(tx *Tx) (string, error) {
	// This would integrate with the actual node/network broadcasting
	// For now, just return a mock transaction hash
	txBytes, err := json.Marshal(tx)
	if err != nil {
		return "", fmt.Errorf("failed to marshal transaction: %w", err)
	}

	// In production, this would send to the network
	fmt.Printf("Broadcasting transaction:\n%s\n", string(txBytes))

	// Return mock hash
	return fmt.Sprintf("0x%x", keys.HashData(txBytes)[:32]), nil
}

// parseEnhancedFee parses fee configuration with intelligent calculation
func (tb *TxBuilder) parseEnhancedFee(config TxConfig, msgs []commontypes.Msg) (Fee, error) {
	fee := Fee{}

	// Parse gas limit
	if config.Gas == "auto" {
		// Use intelligent gas estimation
		gasLimit, err := tb.EstimateGas(msgs)
		if err != nil {
			return fee, fmt.Errorf("failed to estimate gas: %w", err)
		}
		fee.GasLimit = gasLimit
	} else {
		gasLimit, err := strconv.ParseUint(config.Gas, 10, 64)
		if err != nil {
			return fee, fmt.Errorf("invalid gas limit: %w", err)
		}
		fee.GasLimit = gasLimit
	}

	// Use intelligent fee calculation if enabled
	if config.AutoFee && tb.feeConfig.UseIntelligentFee {
		calculatedFee, err := tb.calculateIntelligentFee(config, fee.GasLimit)
		if err != nil {
			return fee, fmt.Errorf("failed to calculate intelligent fee: %w", err)
		}
		fee.Amount = calculatedFee
		return fee, nil
	}

	// Parse fee amount (legacy method)
	if config.Fees != "" {
		coins, err := parseCoins(config.Fees)
		if err != nil {
			return fee, fmt.Errorf("invalid fees: %w", err)
		}
		fee.Amount = coins
	} else if config.GasPrices != "" {
		// Calculate fee from gas prices
		gasPrice, denom, err := parseGasPrice(config.GasPrices)
		if err != nil {
			return fee, fmt.Errorf("invalid gas price: %w", err)
		}

		feeAmount := gasPrice * int64(fee.GasLimit)
		fee.Amount = []commontypes.Coin{
			{
				Denom:  denom,
				Amount: fmt.Sprintf("%d", feeAmount),
			},
		}
	} else {
		// Use default fee calculation
		defaultFee, err := tb.calculateDefaultFee(config, fee.GasLimit)
		if err != nil {
			return fee, fmt.Errorf("failed to calculate default fee: %w", err)
		}
		fee.Amount = defaultFee
	}

	return fee, nil
}

// calculateIntelligentFee calculates fee using intelligent pricing
func (tb *TxBuilder) calculateIntelligentFee(config TxConfig, gasLimit uint64) ([]commontypes.Coin, error) {
	// Determine fee denomination
	feeDenom := config.FeeDenom
	if feeDenom == "" {
		feeDenom = tb.getPreferredFeeDenom()
	}

	// Get gas price based on priority
	gasPrice, err := tb.getGasPriceForPriority(config.FeePriority, feeDenom)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	// Apply fee multiplier
	multiplier := config.FeeMultiplier
	if multiplier == 0 {
		multiplier = 1.0
	}

	// Calculate final fee amount
	feeAmount := int64(float64(gasPrice) * float64(gasLimit) * multiplier)

	// Check against maximum fee limit
	if config.MaxFee != "" {
		maxFeeCoins, err := parseCoins(config.MaxFee)
		if err != nil {
			return nil, fmt.Errorf("invalid max fee: %w", err)
		}

		for _, maxCoin := range maxFeeCoins {
			if maxCoin.Denom == feeDenom {
				maxAmount, err := strconv.ParseInt(maxCoin.Amount, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("invalid max fee amount: %w", err)
				}
				if feeAmount > maxAmount {
					feeAmount = maxAmount
				}
			}
		}
	}

	return []commontypes.Coin{
		{
			Denom:  feeDenom,
			Amount: fmt.Sprintf("%d", feeAmount),
		},
	}, nil
}

// calculateDefaultFee calculates a default fee when no fee is specified
func (tb *TxBuilder) calculateDefaultFee(config TxConfig, gasLimit uint64) ([]commontypes.Coin, error) {
	feeDenom := config.FeeDenom
	if feeDenom == "" {
		feeDenom = tb.getPreferredFeeDenom()
	}

	// Use minimum gas price for the denomination
	var gasPrice int64 = 1 // Default minimum gas price
	for _, denom := range tb.feeConfig.EnabledDenoms {
		if denom.Denom == feeDenom && denom.IsEnabled() {
			// Parse minimum gas price
			amount, _, err := parseAmountDenom(denom.MinGasPrice)
			if err == nil {
				gasPrice = amount
			}
			break
		}
	}

	feeAmount := gasPrice * int64(gasLimit)

	return []commontypes.Coin{
		{
			Denom:  feeDenom,
			Amount: fmt.Sprintf("%d", feeAmount),
		},
	}, nil
}

// getPreferredFeeDenom returns the preferred fee denomination
func (tb *TxBuilder) getPreferredFeeDenom() string {
	if len(tb.feeConfig.EnabledDenoms) == 0 {
		return "ubtc" // Default
	}

	// Return the first enabled denomination with highest priority
	for _, denom := range tb.feeConfig.EnabledDenoms {
		if denom.IsEnabled() {
			return denom.Denom
		}
	}

	return "ubtc" // Fallback
}

// getGasPriceForPriority returns gas price based on priority level
func (tb *TxBuilder) getGasPriceForPriority(priority FeePriority, denom string) (int64, error) {
	// Find fee denomination configuration
	var feeDenom types.FeeDenom
	found := false
	for _, d := range tb.feeConfig.EnabledDenoms {
		if d.Denom == denom {
			feeDenom = d
			found = true
			break
		}
	}

	if !found {
		return 0, fmt.Errorf("fee denomination %s not found", denom)
	}

	// Parse minimum gas price
	minPrice, _, err := parseAmountDenom(feeDenom.MinGasPrice)
	if err != nil {
		return 0, fmt.Errorf("invalid min gas price: %w", err)
	}

	// Calculate price based on priority
	switch priority {
	case FeePriorityLow:
		return minPrice, nil
	case FeePriorityNormal:
		return int64(float64(minPrice) * 1.5), nil
	case FeePriorityHigh:
		return int64(float64(minPrice) * 2.0), nil
	case FeePriorityFast:
		return int64(float64(minPrice) * 3.0), nil
	default:
		return int64(float64(minPrice) * 1.5), nil // Normal priority
	}
}

// parseCoins parses coins string
func parseCoins(coinsStr string) ([]commontypes.Coin, error) {
	// Simple implementation - would be more robust in production
	// Format: "100ubtc" or "100ubtc,50wei"
	if coinsStr == "" {
		return []commontypes.Coin{}, nil
	}

	// For now, just parse single coin
	amount, denom, err := parseAmountDenom(coinsStr)
	if err != nil {
		return nil, err
	}

	return []commontypes.Coin{
		{
			Denom:  denom,
			Amount: fmt.Sprintf("%d", amount),
		},
	}, nil
}

// parseGasPrice parses gas price string
func parseGasPrice(gasPriceStr string) (int64, string, error) {
	// Format: "0.1ubtc"
	amount, denom, err := parseAmountDenom(gasPriceStr)
	if err != nil {
		return 0, "", err
	}
	return amount, denom, nil
}

// parseAmountDenom parses amount and denomination
func parseAmountDenom(str string) (int64, string, error) {
	// Find where numbers end and denom begins
	i := 0
	for i < len(str) && (str[i] >= '0' && str[i] <= '9' || str[i] == '.') {
		i++
	}

	if i == 0 || i == len(str) {
		return 0, "", fmt.Errorf("invalid format: %s", str)
	}

	amountStr := str[:i]
	denom := str[i:]

	// Parse amount (handling decimals by multiplying)
	if amountStr[len(amountStr)-1] == '.' {
		amountStr = amountStr[:len(amountStr)-1]
	}

	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid amount %s: %w", amountStr, err)
	}

	return amount, denom, nil
}

// convertMsgsToSignature converts common types to signature types
func convertMsgsToSignature(msgs []commontypes.Msg) []signature.Message {
	var signatureMsgs []signature.Message
	for _, msg := range msgs {
		signatureMsgs = append(signatureMsgs, signature.Message{
			Type:  fmt.Sprintf("%T", msg),
			Value: msg,
		})
	}
	return signatureMsgs
}

// convertCoinsToSignature converts common types to signature types
func convertCoinsToSignature(coins []commontypes.Coin) []signature.Coin {
	var sigCoins []signature.Coin
	for _, coin := range coins {
		sigCoins = append(sigCoins, signature.Coin{
			Denom:  coin.Denom,
			Amount: coin.Amount,
		})
	}
	return sigCoins
}

// GetKeyring returns the keyring
func (tb *TxBuilder) GetKeyring() keys.Keyring {
	return tb.keyring
}

// Fee optimization and suggestion methods

// SuggestOptimalFee suggests optimal fee for a transaction based on current network conditions
func (tb *TxBuilder) SuggestOptimalFee(msgs []commontypes.Msg, priority FeePriority) (*FeeEstimate, error) {
	// Estimate gas
	gasLimit, err := tb.EstimateGas(msgs)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate gas: %w", err)
	}

	// Get all available fee denominations
	estimates := make([]DenomFeeEstimate, 0, len(tb.feeConfig.EnabledDenoms))

	for _, denom := range tb.feeConfig.EnabledDenoms {
		if !denom.IsEnabled() {
			continue
		}

		gasPrice, err := tb.getGasPriceForPriority(priority, denom.Denom)
		if err != nil {
			continue
		}

		feeAmount := gasPrice * int64(gasLimit)
		estimates = append(estimates, DenomFeeEstimate{
			Denom:     denom.Denom,
			Amount:    fmt.Sprintf("%d", feeAmount),
			GasPrice:  fmt.Sprintf("%d", gasPrice),
			Priority:  denom.Priority,
		})
	}

	return &FeeEstimate{
		GasLimit:    gasLimit,
		Estimates:   estimates,
		Recommended: tb.getRecommendedDenom(estimates),
	}, nil
}

// GetFeeEstimateForDenom gets fee estimate for a specific denomination
func (tb *TxBuilder) GetFeeEstimateForDenom(msgs []commontypes.Msg, denom string, priority FeePriority) (*DenomFeeEstimate, error) {
	gasLimit, err := tb.EstimateGas(msgs)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate gas: %w", err)
	}

	gasPrice, err := tb.getGasPriceForPriority(priority, denom)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	feeAmount := gasPrice * int64(gasLimit)

	// Get denomination priority
	var denomPriority uint32 = 0
	for _, d := range tb.feeConfig.EnabledDenoms {
		if d.Denom == denom {
			denomPriority = d.Priority
			break
		}
	}

	return &DenomFeeEstimate{
		Denom:    denom,
		Amount:   fmt.Sprintf("%d", feeAmount),
		GasPrice: fmt.Sprintf("%d", gasPrice),
		Priority: denomPriority,
	}, nil
}

// GetSupportedFeeDenoms returns list of supported fee denominations
func (tb *TxBuilder) GetSupportedFeeDenoms() []string {
	denoms := make([]string, 0, len(tb.feeConfig.EnabledDenoms))
	for _, denom := range tb.feeConfig.EnabledDenoms {
		if denom.IsEnabled() {
			denoms = append(denoms, denom.Denom)
		}
	}
	return denoms
}

// UpdateFeeConfig updates the fee configuration
func (tb *TxBuilder) UpdateFeeConfig(config *FeeConfig) {
	if config != nil {
		tb.feeConfig = config
	}
}

// GetFeeConfig returns the current fee configuration
func (tb *TxBuilder) GetFeeConfig() *FeeConfig {
	return tb.feeConfig
}

// Helper types for fee estimation

// FeeEstimate contains fee estimates for different denominations
type FeeEstimate struct {
	GasLimit    uint64              `json:"gas_limit"`
	Estimates   []DenomFeeEstimate  `json:"estimates"`
	Recommended string              `json:"recommended"`
}

// DenomFeeEstimate contains fee estimate for a specific denomination
type DenomFeeEstimate struct {
	Denom    string `json:"denom"`
	Amount   string `json:"amount"`
	GasPrice string `json:"gas_price"`
	Priority uint32 `json:"priority"`
}

// getRecommendedDenom returns the recommended denomination based on priority
func (tb *TxBuilder) getRecommendedDenom(estimates []DenomFeeEstimate) string {
	if len(estimates) == 0 {
		return "ubtc"
	}

	// Find denomination with highest priority
	recommended := estimates[0]
	for _, estimate := range estimates[1:] {
		if estimate.Priority > recommended.Priority {
			recommended = estimate
		}
	}

	return recommended.Denom
}

// Default configuration helpers

// getDefaultFeeDenoms returns default fee denominations
func getDefaultFeeDenoms() []types.FeeDenom {
	return []types.FeeDenom{
		types.NewFeeDenom("ubtc", "0.01ubtc", true, 100),
		types.NewFeeDenom("ueth", "100ueth", true, 90),
		types.NewFeeDenom("upulse", "1000upulse", true, 80),
	}
}

// getDefaultDynamicFactors returns default dynamic gas factors
func getDefaultDynamicFactors() types.DynamicGasFactors {
	return types.DynamicGasFactors{
		SizeMultiplier:   1.0,
		ComplexityFactor: 1.0,
		NetworkFactor:    1.0,
		StorageFactor:    1.2,
	}
}

// getDefaultFeeDistribution returns default fee distribution configuration
func getDefaultFeeDistribution() types.FeeDistributionConfig {
	return types.FeeDistributionConfig{
		BurnPercentage:   0.5,  // 50% burned
		ValidatorRewards: 0.3,  // 30% to validators
		CommunityPool:    0.15, // 15% to community pool
		DeveloperFund:    0.05, // 5% to developers
	}
}