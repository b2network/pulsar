package types

// InterfaceRegistry defines the interface for registering interfaces
type InterfaceRegistry interface {
	RegisterInterface(protoName string, iface interface{}, impls ...interface{})
	RegisterImplementations(iface interface{}, impls ...interface{})
}

// RegisterInterfaces registers the fee module's interface types
func RegisterInterfaces(registry InterfaceRegistry) {
	// Register message interfaces
	registry.RegisterInterface(
		"pulsar.fee.v1.Msg",
		(*MsgInterface)(nil),
		&MsgAddFeeDenom{},
		&MsgRemoveFeeDenom{},
		&MsgUpdateFeeDenom{},
		&MsgSetModuleGasConfig{},
		&MsgUpdateGasFactors{},
		&MsgSetFeeDistribution{},
	)

	// Register query interfaces if needed
	registry.RegisterInterface(
		"pulsar.fee.v1.Query",
		(*QueryInterface)(nil),
	)
}

// MsgInterface defines the base interface for fee module messages
type MsgInterface interface {
	ValidateBasic() error
	GetSigners() []string
	Route() string
	Type() string
}

// QueryInterface defines the base interface for fee module queries
type QueryInterface interface {
	// Query-specific methods would go here
}