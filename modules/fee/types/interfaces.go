package types

import (
	keepertypes "github.com/b2network/pulsar/keeper/types"
	commontypes "github.com/b2network/pulsar/types"
)

// GasEstimator defines the interface for gas estimation
type GasEstimator interface {
	// EstimateGas estimates the gas consumption for a set of messages
	EstimateGas(ctx keepertypes.Context, msgs []commontypes.Msg) (uint64, error)

	// EstimateDynamicGas estimates gas with dynamic factors applied
	EstimateDynamicGas(ctx keepertypes.Context, msg commontypes.Msg, factors DynamicGasFactors) (uint64, error)

	// EstimateGasForModule estimates gas for a specific module operation
	EstimateGasForModule(ctx keepertypes.Context, moduleName, msgType string, msg commontypes.Msg) (uint64, error)
}

// FeeCalculator defines the interface for fee calculation
type FeeCalculator interface {
	// CalculateFee calculates the total fee for a transaction
	CalculateFee(ctx keepertypes.Context, req FeeCalculationRequest) (FeeCalculationResponse, error)

	// ValidateFee validates if the provided fee is sufficient
	ValidateFee(ctx keepertypes.Context, msgs []commontypes.Msg, fee commontypes.Coin, gasLimit uint64) error

	// GetMinimumFee returns the minimum fee required for a transaction
	GetMinimumFee(ctx keepertypes.Context, msgs []commontypes.Msg, gasLimit uint64, feeDenom string) (commontypes.Coin, error)
}

// FeeCollector defines the interface for fee collection and distribution
type FeeCollector interface {
	// CollectFees collects fees from a transaction
	CollectFees(ctx keepertypes.Context, fees commontypes.Coin) error

	// DistributeFees distributes collected fees according to configuration
	DistributeFees(ctx keepertypes.Context) error

	// GetCollectedFees returns the amount of fees collected
	GetCollectedFees(ctx keepertypes.Context, denom string) (commontypes.Coin, error)
}

// GasPriceOracle defines the interface for gas price oracle functionality
type GasPriceOracle interface {
	// UpdateGasPrices updates gas price statistics
	UpdateGasPrices(ctx keepertypes.Context) error

	// GetRecommendedGasPrice returns the recommended gas price for a denomination
	GetRecommendedGasPrice(ctx keepertypes.Context, denom string) (string, error)

	// GetGasPriceStats returns gas price statistics for a denomination
	GetGasPriceStats(ctx keepertypes.Context, denom string) (GasPriceStats, error)
}

// AccountKeeper defines the expected interface for the account keeper
type AccountKeeper interface {
	GetAccount(ctx keepertypes.Context, addr string) keepertypes.Account
	SetAccount(ctx keepertypes.Context, acc keepertypes.Account)
	NewAccountWithAddress(ctx keepertypes.Context, addr string) keepertypes.Account
}

// BankKeeper defines the expected interface for the bank keeper
type BankKeeper interface {
	SendCoins(ctx keepertypes.Context, fromAddr, toAddr string, amt commontypes.Coins) error
	SendCoinsFromAccountToModule(ctx keepertypes.Context, senderAddr, recipientModule string, amt commontypes.Coins) error
	SendCoinsFromModuleToAccount(ctx keepertypes.Context, senderModule, recipientAddr string, amt commontypes.Coins) error
	SendCoinsFromModuleToModule(ctx keepertypes.Context, senderModule, recipientModule string, amt commontypes.Coins) error
	BurnCoins(ctx keepertypes.Context, moduleName string, amt commontypes.Coins) error
	GetBalance(ctx keepertypes.Context, addr, denom string) commontypes.Coin
	GetAllBalances(ctx keepertypes.Context, addr string) commontypes.Coins
	SpendableCoins(ctx keepertypes.Context, addr string) commontypes.Coins
}

// StakingKeeper defines the expected interface for the staking keeper
type StakingKeeper interface {
	BondedRatio(ctx keepertypes.Context) float64
	TotalBondedTokens(ctx keepertypes.Context) int64
	GetValidators(ctx keepertypes.Context, maxRetrieve uint32) []keepertypes.Validator
}

// DistributionKeeper defines the expected interface for the distribution keeper
type DistributionKeeper interface {
	AllocateTokensToValidator(ctx keepertypes.Context, val keepertypes.Validator, tokens commontypes.Coins)
	GetFeePool(ctx keepertypes.Context) keepertypes.FeePool
	SetFeePool(ctx keepertypes.Context, feePool keepertypes.FeePool)
}

// Expected interfaces that other modules must implement

// Account represents a user account
type Account interface {
	GetAddress() string
	GetAccountNumber() uint64
	GetSequence() uint64
}

// Validator represents a validator
type Validator interface {
	GetOperatorAddress() string
	GetTokens() int64
	GetCommission() float64
}

// FeePool represents the community fee pool
type FeePool interface {
	GetCommunityPool() commontypes.Coins
	SetCommunityPool(commontypes.Coins)
}