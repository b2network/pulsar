package types

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/b2network/pulsar/types"
)

// FeeDenom represents a denomination that can be used to pay transaction fees
type FeeDenom struct {
	Denom        string `json:"denom"`
	MinGasPrice  string `json:"min_gas_price"`  // Minimum gas price in this denomination
	Enabled      bool   `json:"enabled"`        // Whether this denomination is enabled for fee payment
	Priority     uint32 `json:"priority"`       // Priority for fee payment (higher = preferred)
}

// NewFeeDenom creates a new FeeDenom
func NewFeeDenom(denom, minGasPrice string, enabled bool, priority uint32) FeeDenom {
	return FeeDenom{
		Denom:       denom,
		MinGasPrice: minGasPrice,
		Enabled:     enabled,
		Priority:    priority,
	}
}

// ValidateBasic performs basic validation of FeeDenom
func (fd FeeDenom) ValidateBasic() error {
	if fd.Denom == "" {
		return fmt.Errorf("denom cannot be empty")
	}

	if fd.MinGasPrice == "" {
		return fmt.Errorf("min_gas_price cannot be empty")
	}

	// Validate that min_gas_price is a valid decimal number
	if _, err := strconv.ParseFloat(fd.MinGasPrice, 64); err != nil {
		return fmt.Errorf("invalid min_gas_price: %w", err)
	}

	return nil
}

// IsEnabled returns whether this denomination is enabled for fee payment
func (fd FeeDenom) IsEnabled() bool {
	return fd.Enabled
}

// ModuleGasConfig represents gas configuration for a specific module
type ModuleGasConfig struct {
	ModuleName   string            `json:"module_name"`
	MsgGasRates  map[string]uint64 `json:"msg_gas_rates"`  // Message type to base gas consumption mapping
	BaseGas      uint64            `json:"base_gas"`       // Base gas consumption for any operation in this module
	Enabled      bool              `json:"enabled"`        // Whether gas metering is enabled for this module
}

// NewModuleGasConfig creates a new ModuleGasConfig
func NewModuleGasConfig(moduleName string, baseGas uint64, enabled bool) ModuleGasConfig {
	return ModuleGasConfig{
		ModuleName:  moduleName,
		MsgGasRates: make(map[string]uint64),
		BaseGas:     baseGas,
		Enabled:     enabled,
	}
}

// ValidateBasic performs basic validation of ModuleGasConfig
func (mgc ModuleGasConfig) ValidateBasic() error {
	if mgc.ModuleName == "" {
		return fmt.Errorf("module_name cannot be empty")
	}

	for msgType, gasRate := range mgc.MsgGasRates {
		if msgType == "" {
			return fmt.Errorf("message type cannot be empty")
		}
		if gasRate == 0 {
			return fmt.Errorf("gas rate for message type %s cannot be zero", msgType)
		}
	}

	return nil
}

// GetMsgGasRate returns the gas rate for a specific message type
func (mgc ModuleGasConfig) GetMsgGasRate(msgType string) uint64 {
	if rate, exists := mgc.MsgGasRates[msgType]; exists {
		return rate
	}
	return mgc.BaseGas
}

// SetMsgGasRate sets the gas rate for a specific message type
func (mgc *ModuleGasConfig) SetMsgGasRate(msgType string, gasRate uint64) {
	if mgc.MsgGasRates == nil {
		mgc.MsgGasRates = make(map[string]uint64)
	}
	mgc.MsgGasRates[msgType] = gasRate
}

// DynamicGasFactors represents factors that affect dynamic gas calculation
type DynamicGasFactors struct {
	SizeMultiplier    float64 `json:"size_multiplier"`    // Multiplier based on transaction size
	ComplexityFactor  float64 `json:"complexity_factor"`  // Factor based on operation complexity
	NetworkFactor     float64 `json:"network_factor"`     // Factor based on network congestion
	StorageFactor     float64 `json:"storage_factor"`     // Factor based on storage operations
}

// NewDynamicGasFactors creates a new DynamicGasFactors with default values
func NewDynamicGasFactors() DynamicGasFactors {
	return DynamicGasFactors{
		SizeMultiplier:   1.0,
		ComplexityFactor: 1.0,
		NetworkFactor:    1.0,
		StorageFactor:    1.0,
	}
}

// ValidateBasic performs basic validation of DynamicGasFactors
func (dgf DynamicGasFactors) ValidateBasic() error {
	if dgf.SizeMultiplier < 0 {
		return fmt.Errorf("size_multiplier cannot be negative")
	}
	if dgf.ComplexityFactor < 0 {
		return fmt.Errorf("complexity_factor cannot be negative")
	}
	if dgf.NetworkFactor < 0 {
		return fmt.Errorf("network_factor cannot be negative")
	}
	if dgf.StorageFactor < 0 {
		return fmt.Errorf("storage_factor cannot be negative")
	}
	return nil
}

// FeeCalculationRequest represents a request to calculate transaction fees
type FeeCalculationRequest struct {
	Messages     []types.Msg  `json:"messages"`
	GasLimit     uint64       `json:"gas_limit"`
	GasPrice     string       `json:"gas_price"`
	FeeDenom     string       `json:"fee_denom"`
	Simulation   bool         `json:"simulation"`
}

// ValidateBasic performs basic validation of FeeCalculationRequest
func (fcr FeeCalculationRequest) ValidateBasic() error {
	if len(fcr.Messages) == 0 {
		return fmt.Errorf("messages cannot be empty")
	}

	if fcr.GasPrice == "" {
		return fmt.Errorf("gas_price cannot be empty")
	}

	if fcr.FeeDenom == "" {
		return fmt.Errorf("fee_denom cannot be empty")
	}

	// Validate gas price format
	if _, err := strconv.ParseFloat(fcr.GasPrice, 64); err != nil {
		return fmt.Errorf("invalid gas_price format: %w", err)
	}

	return nil
}

// FeeCalculationResponse represents the response from fee calculation
type FeeCalculationResponse struct {
	EstimatedGas      uint64      `json:"estimated_gas"`
	MinGasPrice       string      `json:"min_gas_price"`
	RecommendedGasPrice string    `json:"recommended_gas_price"`
	TotalFee          types.Coin  `json:"total_fee"`
	BreakdownByModule []GasBreakdown `json:"breakdown_by_module"`
}

// GasBreakdown represents gas usage breakdown by module/operation
type GasBreakdown struct {
	ModuleName   string `json:"module_name"`
	Operation    string `json:"operation"`
	BaseGas      uint64 `json:"base_gas"`
	DynamicGas   uint64 `json:"dynamic_gas"`
	TotalGas     uint64 `json:"total_gas"`
}

// GasPriceStats represents statistical information about gas prices
type GasPriceStats struct {
	Denom           string            `json:"denom"`
	MinPrice        string            `json:"min_price"`
	MaxPrice        string            `json:"max_price"`
	AveragePrice    string            `json:"average_price"`
	MedianPrice     string            `json:"median_price"`
	RecommendedPrice string           `json:"recommended_price"`
	UpdateTime      int64             `json:"update_time"`
	SampleSize      uint64            `json:"sample_size"`
}

// ValidateBasic performs basic validation of GasPriceStats
func (gps GasPriceStats) ValidateBasic() error {
	if gps.Denom == "" {
		return fmt.Errorf("denom cannot be empty")
	}

	prices := []string{gps.MinPrice, gps.MaxPrice, gps.AveragePrice, gps.MedianPrice, gps.RecommendedPrice}
	for i, price := range prices {
		if price == "" {
			continue // Some prices might be empty initially
		}
		if _, err := strconv.ParseFloat(price, 64); err != nil {
			return fmt.Errorf("invalid price format at index %d: %w", i, err)
		}
	}

	return nil
}

// FeeDistributionConfig represents configuration for fee distribution
type FeeDistributionConfig struct {
	ValidatorRewards   float64 `json:"validator_rewards"`   // Percentage to validators (0.0-1.0)
	CommunityPool      float64 `json:"community_pool"`      // Percentage to community pool (0.0-1.0)
	BurnPercentage     float64 `json:"burn_percentage"`     // Percentage to burn (0.0-1.0)
	DeveloperFund      float64 `json:"developer_fund"`      // Percentage to developer fund (0.0-1.0)
}

// ValidateBasic performs basic validation of FeeDistributionConfig
func (fdc FeeDistributionConfig) ValidateBasic() error {
	total := fdc.ValidatorRewards + fdc.CommunityPool + fdc.BurnPercentage + fdc.DeveloperFund

	if total != 1.0 {
		return fmt.Errorf("fee distribution percentages must sum to 1.0, got %.6f", total)
	}

	percentages := []float64{fdc.ValidatorRewards, fdc.CommunityPool, fdc.BurnPercentage, fdc.DeveloperFund}
	for i, percentage := range percentages {
		if percentage < 0 || percentage > 1 {
			return fmt.Errorf("percentage at index %d must be between 0.0 and 1.0, got %.6f", i, percentage)
		}
	}

	return nil
}

// DefaultFeeDistributionConfig returns the default fee distribution configuration
func DefaultFeeDistributionConfig() FeeDistributionConfig {
	return FeeDistributionConfig{
		ValidatorRewards: 0.7,   // 70% to validators
		CommunityPool:    0.2,   // 20% to community pool
		BurnPercentage:   0.05,  // 5% burned
		DeveloperFund:    0.05,  // 5% to developer fund
	}
}

// ParseGasPrice parses a gas price string in the format "0.1ubtc"
func ParseGasPrice(gasPriceStr string) (price float64, denom string, err error) {
	gasPriceStr = strings.TrimSpace(gasPriceStr)
	if gasPriceStr == "" {
		return 0, "", fmt.Errorf("empty gas price string")
	}

	// Find where the numeric part ends and denom begins
	i := 0
	for i < len(gasPriceStr) && (gasPriceStr[i] >= '0' && gasPriceStr[i] <= '9' || gasPriceStr[i] == '.') {
		i++
	}

	if i == 0 || i == len(gasPriceStr) {
		return 0, "", fmt.Errorf("invalid gas price format: %s", gasPriceStr)
	}

	priceStr := gasPriceStr[:i]
	denom = gasPriceStr[i:]

	price, err = strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid price in gas price string: %w", err)
	}

	if price < 0 {
		return 0, "", fmt.Errorf("gas price cannot be negative")
	}

	return price, denom, nil
}

// FormatGasPrice formats a gas price from components
func FormatGasPrice(price float64, denom string) string {
	return fmt.Sprintf("%.9f%s", price, denom)
}