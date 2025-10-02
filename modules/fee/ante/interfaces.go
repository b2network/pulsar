package ante

import (
	"fmt"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	commontypes "github.com/b2network/pulsar/types"
)

// AccountKeeper defines the expected account keeper interface
type AccountKeeper interface {
	GetAccount(ctx keepertypes.Context, addr []byte) keepertypes.Account
	SetAccount(ctx keepertypes.Context, acc keepertypes.Account)
	GetModuleAddress(moduleName string) []byte
}

// BankKeeper defines the expected bank keeper interface
type BankKeeper interface {
	GetAllBalances(ctx keepertypes.Context, addr []byte) keepertypes.Coins
	GetBalance(ctx keepertypes.Context, addr []byte, denom string) keepertypes.Coin
	SendCoinsFromAccountToModule(ctx keepertypes.Context, senderAddr []byte, recipientModule string, amt keepertypes.Coins) error
	SendCoinsFromModuleToAccount(ctx keepertypes.Context, senderModule string, recipientAddr []byte, amt keepertypes.Coins) error
	SendCoinsFromModuleToModule(ctx keepertypes.Context, senderModule, recipientModule string, amt keepertypes.Coins) error
	BurnCoins(ctx keepertypes.Context, moduleName string, amounts keepertypes.Coins) error
}

// AnteHandlerFunc defines the signature for ante handler functions
type AnteHandlerFunc func(ctx keepertypes.Context, tx Tx, simulate bool) (newCtx keepertypes.Context, err error)

// AnteDecorator wraps the next AnteHandler to perform custom pre-processing
type AnteDecorator interface {
	AnteHandle(ctx keepertypes.Context, tx Tx, simulate bool, next AnteHandlerFunc) (newCtx keepertypes.Context, err error)
}

// Tx defines the interface that all transactions must implement
type Tx interface {
	// GetMsgs returns the messages in the transaction
	GetMsgs() []commontypes.Msg

	// ValidateBasic performs basic validation of the transaction
	ValidateBasic() error
}

// FeeTx defines the interface for transactions that can pay fees
type FeeTx interface {
	Tx

	// GetGas returns the gas limit for the transaction
	GetGas() uint64

	// GetFee returns the fee amount for the transaction
	GetFee() keepertypes.Coins

	// FeePayer returns the address that will pay the fees
	FeePayer() []byte

	// FeeGranter returns the address that granted permission to pay fees (if any)
	FeeGranter() []byte
}

// GasMeter defines the interface for gas metering
type GasMeter interface {
	// GasConsumed returns the amount of gas consumed
	GasConsumed() uint64

	// GasConsumedToLimit returns the amount of gas consumed up to the limit
	GasConsumedToLimit() uint64

	// Limit returns the gas limit
	Limit() uint64

	// ConsumeGas consumes the given amount of gas
	ConsumeGas(amount uint64, descriptor string)

	// RefundGas refunds the given amount of gas
	RefundGas(amount uint64, descriptor string)

	// IsPastLimit returns true if gas consumed is past the limit
	IsPastLimit() bool

	// IsOutOfGas returns true if gas consumed is >= limit
	IsOutOfGas() bool

	// String returns a string representation
	String() string
}

// Context defines the extended context interface for ante handlers
type Context interface {
	keepertypes.Context

	// GasMeter returns the gas meter
	GasMeter() GasMeter

	// WithGasMeter returns a new context with the given gas meter
	WithGasMeter(meter GasMeter) Context

	// WithPriority returns a new context with the given priority
	WithPriority(priority int64) Context

	// Priority returns the transaction priority
	Priority() int64

	// IsCheckTx returns true if this is a CheckTx context
	IsCheckTx() bool

	// IsReCheckTx returns true if this is a ReCheckTx context
	IsReCheckTx() bool

	// IsDeliverTx returns true if this is a DeliverTx context
	IsDeliverTx() bool

	// TxBytes returns the transaction bytes
	TxBytes() ([]byte, error)

	// WithValue stores a value in the context
	WithValue(key interface{}, value interface{}) Context

	// Value retrieves a value from the context
	Value(key interface{}) interface{}
}

// BasicGasMeter implements the GasMeter interface
type BasicGasMeter struct {
	limit    uint64
	consumed uint64
}

// NewGasMeter creates a new gas meter with the given limit
func NewGasMeter(limit uint64) GasMeter {
	return &BasicGasMeter{
		limit:    limit,
		consumed: 0,
	}
}

// NewInfiniteGasMeter creates a gas meter with unlimited gas
func NewInfiniteGasMeter() GasMeter {
	return &BasicGasMeter{
		limit:    ^uint64(0), // Max uint64
		consumed: 0,
	}
}

// GasConsumed returns the amount of gas consumed
func (gm *BasicGasMeter) GasConsumed() uint64 {
	return gm.consumed
}

// GasConsumedToLimit returns the amount of gas consumed up to the limit
func (gm *BasicGasMeter) GasConsumedToLimit() uint64 {
	if gm.consumed > gm.limit {
		return gm.limit
	}
	return gm.consumed
}

// Limit returns the gas limit
func (gm *BasicGasMeter) Limit() uint64 {
	return gm.limit
}

// ConsumeGas consumes the given amount of gas
func (gm *BasicGasMeter) ConsumeGas(amount uint64, descriptor string) {
	gm.consumed += amount
	if gm.consumed > gm.limit {
		panic(ErrorOutOfGas{Descriptor: descriptor})
	}
}

// RefundGas refunds the given amount of gas
func (gm *BasicGasMeter) RefundGas(amount uint64, descriptor string) {
	if gm.consumed >= amount {
		gm.consumed -= amount
	} else {
		gm.consumed = 0
	}
}

// IsPastLimit returns true if gas consumed is past the limit
func (gm *BasicGasMeter) IsPastLimit() bool {
	return gm.consumed > gm.limit
}

// IsOutOfGas returns true if gas consumed is >= limit
func (gm *BasicGasMeter) IsOutOfGas() bool {
	return gm.consumed >= gm.limit
}

// String returns a string representation
func (gm *BasicGasMeter) String() string {
	return fmt.Sprintf("BasicGasMeter{limit: %d, consumed: %d}", gm.limit, gm.consumed)
}

// ErrorOutOfGas defines an error for when the gas meter runs out of gas
type ErrorOutOfGas struct {
	Descriptor string
}

// Error implements the error interface
func (e ErrorOutOfGas) Error() string {
	return fmt.Sprintf("out of gas: %s", e.Descriptor)
}

// ExtendedContext extends the basic context with ante handler functionality
type ExtendedContext struct {
	keepertypes.Context
	gasMeter   GasMeter
	priority   int64
	isCheckTx  bool
	isDeliverTx bool
	txBytes    []byte
	values     map[interface{}]interface{}
}

// NewExtendedContext creates a new extended context
func NewExtendedContext(base keepertypes.Context, gasMeter GasMeter) Context {
	return &ExtendedContext{
		Context:  base,
		gasMeter: gasMeter,
		priority: 0,
		values:   make(map[interface{}]interface{}),
	}
}

// GasMeter returns the gas meter
func (ctx *ExtendedContext) GasMeter() GasMeter {
	return ctx.gasMeter
}

// WithGasMeter returns a new context with the given gas meter
func (ctx *ExtendedContext) WithGasMeter(meter GasMeter) Context {
	newCtx := *ctx
	newCtx.gasMeter = meter
	return &newCtx
}

// WithPriority returns a new context with the given priority
func (ctx *ExtendedContext) WithPriority(priority int64) Context {
	newCtx := *ctx
	newCtx.priority = priority
	return &newCtx
}

// Priority returns the transaction priority
func (ctx *ExtendedContext) Priority() int64 {
	return ctx.priority
}

// IsCheckTx returns true if this is a CheckTx context
func (ctx *ExtendedContext) IsCheckTx() bool {
	return ctx.isCheckTx
}

// IsReCheckTx returns true if this is a ReCheckTx context
func (ctx *ExtendedContext) IsReCheckTx() bool {
	return false // Not implemented in this basic version
}

// IsDeliverTx returns true if this is a DeliverTx context
func (ctx *ExtendedContext) IsDeliverTx() bool {
	return ctx.isDeliverTx
}

// TxBytes returns the transaction bytes
func (ctx *ExtendedContext) TxBytes() ([]byte, error) {
	if ctx.txBytes == nil {
		return nil, fmt.Errorf("transaction bytes not set")
	}
	return ctx.txBytes, nil
}

// WithValue stores a value in the context
func (ctx *ExtendedContext) WithValue(key interface{}, value interface{}) Context {
	newCtx := *ctx
	newValues := make(map[interface{}]interface{})
	for k, v := range ctx.values {
		newValues[k] = v
	}
	newValues[key] = value
	newCtx.values = newValues
	return &newCtx
}

// Value retrieves a value from the context
func (ctx *ExtendedContext) Value(key interface{}) interface{} {
	return ctx.values[key]
}

// ChainAnteDecorators chains ante decorators together
func ChainAnteDecorators(decorators ...AnteDecorator) AnteHandlerFunc {
	if len(decorators) == 0 {
		return func(ctx keepertypes.Context, tx Tx, simulate bool) (keepertypes.Context, error) {
			return ctx, nil
		}
	}

	if len(decorators) == 1 {
		return func(ctx keepertypes.Context, tx Tx, simulate bool) (keepertypes.Context, error) {
			return decorators[0].AnteHandle(ctx, tx, simulate, func(ctx keepertypes.Context, tx Tx, simulate bool) (keepertypes.Context, error) {
				return ctx, nil
			})
		}
	}

	return func(ctx keepertypes.Context, tx Tx, simulate bool) (keepertypes.Context, error) {
		return decorators[0].AnteHandle(ctx, tx, simulate, ChainAnteDecorators(decorators[1:]...))
	}
}