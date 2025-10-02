package types

// Module constants
const (
	// ModuleName defines the module name
	ModuleName = "fee"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_fee"
)

// Store prefixes
var (
	// FeeDenomKeyPrefix is the prefix for fee denomination storage
	FeeDenomKeyPrefix = []byte{0x01}

	// ModuleGasConfigKeyPrefix is the prefix for module gas configuration storage
	ModuleGasConfigKeyPrefix = []byte{0x02}

	// DynamicGasFactorsKey is the key for dynamic gas factors storage
	DynamicGasFactorsKey = []byte{0x03}

	// FeeDistributionConfigKey is the key for fee distribution configuration storage
	FeeDistributionConfigKey = []byte{0x04}

	// GasPriceStatsKeyPrefix is the prefix for gas price statistics storage
	GasPriceStatsKeyPrefix = []byte{0x05}

	// TotalFeesCollectedKey is the key for total fees collected counter
	TotalFeesCollectedKey = []byte{0x06}

	// AverageGasPerTxKey is the key for average gas per transaction storage
	AverageGasPerTxKey = []byte{0x07}

	// NetworkCongestionKey is the key for network congestion metrics
	NetworkCongestionKey = []byte{0x08}
)

// Module account names
const (
	// FeeCollectorName is the name of the fee collector module account
	FeeCollectorName = "fee_collector"

	// StakingRewardsName is the name of the staking rewards module account
	StakingRewardsName = "bonded_tokens_pool"

	// CommunityPoolName is the name of the community pool module account
	CommunityPoolName = "distribution"

	// DeveloperFundName is the name of the developer fund module account
	DeveloperFundName = "developer_fund"
)

// Event types and attributes
const (
	// EventTypeFeePayment is the event type for fee payments
	EventTypeFeePayment = "fee_payment"

	// EventTypeGasRefund is the event type for gas refunds
	EventTypeGasRefund = "gas_refund"

	// EventTypeFeeDistribution is the event type for fee distribution
	EventTypeFeeDistribution = "fee_distribution"

	// EventTypeGasConfigUpdate is the event type for gas configuration updates
	EventTypeGasConfigUpdate = "gas_config_update"

	// EventTypeDenomUpdate is the event type for denomination updates
	EventTypeDenomUpdate = "denom_update"
)

// Event attributes
const (
	// AttributeKeyFeePayer is the key for fee payer address
	AttributeKeyFeePayer = "fee_payer"

	// AttributeKeyFees is the key for fee amounts
	AttributeKeyFees = "fees"

	// AttributeKeyGasLimit is the key for gas limit
	AttributeKeyGasLimit = "gas_limit"

	// AttributeKeyRefundRecipient is the key for refund recipient
	AttributeKeyRefundRecipient = "refund_recipient"

	// AttributeKeyRefundAmount is the key for refund amount
	AttributeKeyRefundAmount = "refund_amount"

	// AttributeKeyUnusedGas is the key for unused gas amount
	AttributeKeyUnusedGas = "unused_gas"

	// AttributeKeyModule is the key for module name
	AttributeKeyModule = "module"

	// AttributeKeyMessageType is the key for message type
	AttributeKeyMessageType = "message_type"

	// AttributeKeyGasAmount is the key for gas amount
	AttributeKeyGasAmount = "gas_amount"

	// AttributeKeyDenom is the key for denomination
	AttributeKeyDenom = "denom"

	// AttributeKeyMinGasPrice is the key for minimum gas price
	AttributeKeyMinGasPrice = "min_gas_price"

	// AttributeKeyPriority is the key for priority
	AttributeKeyPriority = "priority"

	// AttributeKeyEnabled is the key for enabled status
	AttributeKeyEnabled = "enabled"

	// AttributeKeyBurnAmount is the key for burn amount
	AttributeKeyBurnAmount = "burn_amount"

	// AttributeKeyValidatorAmount is the key for validator rewards amount
	AttributeKeyValidatorAmount = "validator_amount"

	// AttributeKeyCommunityAmount is the key for community pool amount
	AttributeKeyCommunityAmount = "community_amount"

	// AttributeKeyDeveloperAmount is the key for developer fund amount
	AttributeKeyDeveloperAmount = "developer_amount"
)

// Context keys for storing values in the context
const (
	// ContextKeyDynamicFactors is used to store dynamic gas factors in context
	ContextKeyDynamicFactors = "dynamic_gas_factors"

	// ContextKeyGasProfiler is used to store gas profiler in context
	ContextKeyGasProfiler = "gas_profiler"

	// ContextKeyFeeCalculator is used to store fee calculator in context
	ContextKeyFeeCalculator = "fee_calculator"
)

// Default configuration values
const (
	// DefaultMaxGasWanted is the default maximum gas wanted per transaction
	DefaultMaxGasWanted uint64 = 1000000

	// DefaultMinGasPrice is the default minimum gas price
	DefaultMinGasPrice = "0.01ubtc"

	// DefaultProfilingEnabled indicates if gas profiling is enabled by default
	DefaultProfilingEnabled = false

	// DefaultGasRefundEnabled indicates if gas refunds are enabled by default
	DefaultGasRefundEnabled = true

	// DefaultBaseTxGas is the default base gas for any transaction
	DefaultBaseTxGas uint64 = 10000

	// DefaultSizeMultiplier is the default size multiplier for dynamic gas calculation
	DefaultSizeMultiplier = 1.0

	// DefaultComplexityFactor is the default complexity factor for dynamic gas calculation
	DefaultComplexityFactor = 1.0

	// DefaultNetworkFactor is the default network factor for dynamic gas calculation
	DefaultNetworkFactor = 1.0

	// DefaultStorageFactor is the default storage factor for dynamic gas calculation
	DefaultStorageFactor = 1.0
)

// Gas limits for various operations
const (
	// MaxGasPerTx is the maximum gas allowed per transaction
	MaxGasPerTx uint64 = 10000000

	// MinGasPerTx is the minimum gas required per transaction
	MinGasPerTx uint64 = 5000

	// GasPerByte is the gas cost per byte of transaction data
	GasPerByte uint64 = 10

	// GasPerSignature is the gas cost per signature verification
	GasPerSignature uint64 = 1000
)

// Priority levels
const (
	// PriorityLow represents low priority transactions
	PriorityLow int64 = 1000

	// PriorityNormal represents normal priority transactions
	PriorityNormal int64 = 5000

	// PriorityHigh represents high priority transactions
	PriorityHigh int64 = 10000

	// PriorityUrgent represents urgent priority transactions
	PriorityUrgent int64 = 20000
)

// Fee multipliers by priority
const (
	// LowPriorityMultiplier is the fee multiplier for low priority
	LowPriorityMultiplier = 1.0

	// NormalPriorityMultiplier is the fee multiplier for normal priority
	NormalPriorityMultiplier = 1.5

	// HighPriorityMultiplier is the fee multiplier for high priority
	HighPriorityMultiplier = 2.0

	// UrgentPriorityMultiplier is the fee multiplier for urgent priority
	UrgentPriorityMultiplier = 3.0
)