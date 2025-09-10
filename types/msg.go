package types

// Msg defines the interface for transaction messages
type Msg interface {
	// ValidateBasic performs basic validation of the message
	ValidateBasic() error

	// GetSigners returns the addresses that must sign the transaction
	GetSigners() [][]byte
}

// Event represents a blockchain event
type Event struct {
	Type       string      `json:"type"`
	Attributes []Attribute `json:"attributes"`
}

// Attribute represents an event attribute
type Attribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// NewEvent creates a new event
func NewEvent(eventType string, attributes ...Attribute) Event {
	return Event{
		Type:       eventType,
		Attributes: attributes,
	}
}

// NewAttribute creates a new attribute
func NewAttribute(key, value string) Attribute {
	return Attribute{
		Key:   key,
		Value: value,
	}
}
