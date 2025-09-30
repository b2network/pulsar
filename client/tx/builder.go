package tx

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/b2network/pulsar/crypto/keys"
	"github.com/b2network/pulsar/crypto/signature"
	commontypes "github.com/b2network/pulsar/types"
)

// TxBuilder handles transaction construction, signing, and broadcasting
type TxBuilder struct {
	chainID  string
	keyring  keys.Keyring
	signer   *signature.Signer
	verifier *signature.Verifier
}

// NewTxBuilder creates a new transaction builder
func NewTxBuilder(chainID, keyDir string) (*TxBuilder, error) {
	keyring, err := keys.NewFileKeyring(keyDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create keyring: %w", err)
	}

	signer := signature.NewSigner(chainID, keyring)
	verifier := signature.NewVerifier(chainID)

	return &TxBuilder{
		chainID:  chainID,
		keyring:  keyring,
		signer:   signer,
		verifier: verifier,
	}, nil
}

// TxConfig contains transaction configuration
type TxConfig struct {
	From         string // Key name or address
	Fees         string // Fee amount, e.g., "100ubtc"
	Gas          string // Gas limit, "auto" for automatic calculation
	GasPrices    string // Gas prices, e.g., "0.1ubtc"
	Memo         string // Transaction memo
	AccountNum   uint64 // Account number
	Sequence     uint64 // Account sequence
	TimeoutHeight uint64 // Block timeout height
}

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

	// Parse fee
	fee, err := tb.parseFee(config.Fees, config.Gas, config.GasPrices)
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

// EstimateGas estimates gas for a transaction
func (tb *TxBuilder) EstimateGas(msgs []commontypes.Msg) (uint64, error) {
	// Simple gas estimation based on message count and type
	// In production, this would simulate the transaction
	baseGas := uint64(50000)
	msgGas := uint64(10000) * uint64(len(msgs))

	return baseGas + msgGas, nil
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

// parseFee parses fee configuration
func (tb *TxBuilder) parseFee(feesStr, gasStr, gasPricesStr string) (Fee, error) {
	fee := Fee{}

	// Parse gas limit
	if gasStr == "auto" {
		fee.GasLimit = 200000 // Default gas limit
	} else {
		gasLimit, err := strconv.ParseUint(gasStr, 10, 64)
		if err != nil {
			return fee, fmt.Errorf("invalid gas limit: %w", err)
		}
		fee.GasLimit = gasLimit
	}

	// Parse fee amount
	if feesStr != "" {
		coins, err := parseCoins(feesStr)
		if err != nil {
			return fee, fmt.Errorf("invalid fees: %w", err)
		}
		fee.Amount = coins
	} else if gasPricesStr != "" {
		// Calculate fee from gas prices
		gasPrice, denom, err := parseGasPrice(gasPricesStr)
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
	}

	return fee, nil
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