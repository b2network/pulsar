package types

import (
	"fmt"
	"time"
)

// Parameter keys for the fee module
var (
	KeyMaxGasWanted                = []byte("MaxGasWanted")
	KeyTxSizeCostPerByte          = []byte("TxSizeCostPerByte")
	KeySigVerifyCostED25519       = []byte("SigVerifyCostED25519")
	KeySigVerifyCostSecp256k1     = []byte("SigVerifyCostSecp256k1")
	KeyGasPriceUpdateInterval     = []byte("GasPriceUpdateInterval")
	KeyMinGasPriceMultiplier      = []byte("MinGasPriceMultiplier")
	KeyMaxGasPriceMultiplier      = []byte("MaxGasPriceMultiplier")
	KeyNetworkCongestionThreshold = []byte("NetworkCongestionThreshold")
	KeyFeeCollectionEnabled       = []byte("FeeCollectionEnabled")
)

// Params defines the parameters for the fee module
type Params struct {
	MaxGasWanted                uint64        `json:"max_gas_wanted"`
	TxSizeCostPerByte          uint64        `json:"tx_size_cost_per_byte"`
	SigVerifyCostED25519       uint64        `json:"sig_verify_cost_ed25519"`
	SigVerifyCostSecp256k1     uint64        `json:"sig_verify_cost_secp256k1"`
	GasPriceUpdateInterval     time.Duration `json:"gas_price_update_interval"`
	MinGasPriceMultiplier      float64       `json:"min_gas_price_multiplier"`
	MaxGasPriceMultiplier      float64       `json:"max_gas_price_multiplier"`
	NetworkCongestionThreshold float64       `json:"network_congestion_threshold"`
	FeeCollectionEnabled       bool          `json:"fee_collection_enabled"`
}

// NewParams creates a new Params instance
func NewParams(
	maxGasWanted, txSizeCostPerByte, sigVerifyCostED25519, sigVerifyCostSecp256k1 uint64,
	gasPriceUpdateInterval time.Duration,
	minGasPriceMultiplier, maxGasPriceMultiplier, networkCongestionThreshold float64,
	feeCollectionEnabled bool,
) Params {
	return Params{
		MaxGasWanted:                maxGasWanted,
		TxSizeCostPerByte:          txSizeCostPerByte,
		SigVerifyCostED25519:       sigVerifyCostED25519,
		SigVerifyCostSecp256k1:     sigVerifyCostSecp256k1,
		GasPriceUpdateInterval:     gasPriceUpdateInterval,
		MinGasPriceMultiplier:      minGasPriceMultiplier,
		MaxGasPriceMultiplier:      maxGasPriceMultiplier,
		NetworkCongestionThreshold: networkCongestionThreshold,
		FeeCollectionEnabled:       feeCollectionEnabled,
	}
}

// DefaultParams returns default parameters for the fee module
func DefaultParams() Params {
	return NewParams(
		200000,                // MaxGasWanted: 200k gas
		10,                    // TxSizeCostPerByte: 10 gas per byte
		590,                   // SigVerifyCostED25519: 590 gas
		1000,                  // SigVerifyCostSecp256k1: 1000 gas
		10*time.Minute,        // GasPriceUpdateInterval: 10 minutes
		0.8,                   // MinGasPriceMultiplier: 80% of average
		2.0,                   // MaxGasPriceMultiplier: 200% of average
		0.75,                  // NetworkCongestionThreshold: 75%
		true,                  // FeeCollectionEnabled: true
	)
}

// ValidateBasic performs basic validation of parameters
func (p Params) ValidateBasic() error {
	if p.MaxGasWanted == 0 {
		return fmt.Errorf("max gas wanted must be positive")
	}

	if p.TxSizeCostPerByte == 0 {
		return fmt.Errorf("tx size cost per byte must be positive")
	}

	if p.SigVerifyCostED25519 == 0 {
		return fmt.Errorf("signature verification cost for ed25519 must be positive")
	}

	if p.SigVerifyCostSecp256k1 == 0 {
		return fmt.Errorf("signature verification cost for secp256k1 must be positive")
	}

	if p.GasPriceUpdateInterval <= 0 {
		return fmt.Errorf("gas price update interval must be positive")
	}

	if p.MinGasPriceMultiplier <= 0 || p.MinGasPriceMultiplier > 1 {
		return fmt.Errorf("min gas price multiplier must be between 0 and 1")
	}

	if p.MaxGasPriceMultiplier <= 1 {
		return fmt.Errorf("max gas price multiplier must be greater than 1")
	}

	if p.NetworkCongestionThreshold <= 0 || p.NetworkCongestionThreshold > 1 {
		return fmt.Errorf("network congestion threshold must be between 0 and 1")
	}

	return nil
}

// String returns a human-readable string representation of the parameters
func (p Params) String() string {
	return fmt.Sprintf(`Fee Params:
  Max Gas Wanted: %d
  Tx Size Cost Per Byte: %d
  Sig Verify Cost ED25519: %d
  Sig Verify Cost Secp256k1: %d
  Gas Price Update Interval: %s
  Min Gas Price Multiplier: %.2f
  Max Gas Price Multiplier: %.2f
  Network Congestion Threshold: %.2f
  Fee Collection Enabled: %t`,
		p.MaxGasWanted,
		p.TxSizeCostPerByte,
		p.SigVerifyCostED25519,
		p.SigVerifyCostSecp256k1,
		p.GasPriceUpdateInterval,
		p.MinGasPriceMultiplier,
		p.MaxGasPriceMultiplier,
		p.NetworkCongestionThreshold,
		p.FeeCollectionEnabled,
	)
}