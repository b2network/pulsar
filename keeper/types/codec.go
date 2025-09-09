package types

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// JSONCodec implements Codec using JSON encoding
type JSONCodec struct{}

// NewJSONCodec creates a new JSONCodec
func NewJSONCodec() Codec {
	return &JSONCodec{}
}

// Marshal encodes an object to bytes using JSON
func (c *JSONCodec) Marshal(obj interface{}) ([]byte, error) {
	if obj == nil {
		return nil, fmt.Errorf("cannot marshal nil object")
	}

	return json.Marshal(obj)
}

// Unmarshal decodes bytes to an object using JSON
func (c *JSONCodec) Unmarshal(bz []byte, obj interface{}) error {
	if len(bz) == 0 {
		return fmt.Errorf("cannot unmarshal empty bytes")
	}

	if obj == nil {
		return fmt.Errorf("cannot unmarshal to nil object")
	}

	return json.Unmarshal(bz, obj)
}

// MarshalJSON encodes an object to JSON (same as Marshal for JSONCodec)
func (c *JSONCodec) MarshalJSON(obj interface{}) ([]byte, error) {
	return c.Marshal(obj)
}

// UnmarshalJSON decodes JSON to an object (same as Unmarshal for JSONCodec)
func (c *JSONCodec) UnmarshalJSON(bz []byte, obj interface{}) error {
	return c.Unmarshal(bz, obj)
}

// ProtoCodec implements Codec using Protocol Buffers (placeholder)
type ProtoCodec struct{}

// NewProtoCodec creates a new ProtoCodec
func NewProtoCodec() Codec {
	return &ProtoCodec{}
}

// Marshal encodes an object to bytes using Protocol Buffers
func (c *ProtoCodec) Marshal(obj interface{}) ([]byte, error) {
	// In a real implementation, this would use protobuf marshaling
	// For now, fall back to JSON
	return json.Marshal(obj)
}

// Unmarshal decodes bytes to an object using Protocol Buffers
func (c *ProtoCodec) Unmarshal(bz []byte, obj interface{}) error {
	// In a real implementation, this would use protobuf unmarshaling
	// For now, fall back to JSON
	return json.Unmarshal(bz, obj)
}

// MarshalJSON encodes an object to JSON
func (c *ProtoCodec) MarshalJSON(obj interface{}) ([]byte, error) {
	return json.Marshal(obj)
}

// UnmarshalJSON decodes JSON to an object
func (c *ProtoCodec) UnmarshalJSON(bz []byte, obj interface{}) error {
	return json.Unmarshal(bz, obj)
}

// TypedCodec adds type information to encoded data
type TypedCodec struct {
	inner Codec
}

// NewTypedCodec creates a new TypedCodec
func NewTypedCodec(inner Codec) Codec {
	return &TypedCodec{inner: inner}
}

// TypedData wraps data with type information
type TypedData struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Marshal encodes an object with type information
func (c *TypedCodec) Marshal(obj interface{}) ([]byte, error) {
	if obj == nil {
		return nil, fmt.Errorf("cannot marshal nil object")
	}

	typedData := TypedData{
		Type: reflect.TypeOf(obj).String(),
		Data: obj,
	}

	return c.inner.Marshal(typedData)
}

// Unmarshal decodes bytes with type verification
func (c *TypedCodec) Unmarshal(bz []byte, obj interface{}) error {
	if len(bz) == 0 {
		return fmt.Errorf("cannot unmarshal empty bytes")
	}

	if obj == nil {
		return fmt.Errorf("cannot unmarshal to nil object")
	}

	var typedData TypedData
	if err := c.inner.Unmarshal(bz, &typedData); err != nil {
		return fmt.Errorf("failed to unmarshal typed data: %w", err)
	}

	expectedType := reflect.TypeOf(obj).String()
	if typedData.Type != expectedType {
		return fmt.Errorf("type mismatch: expected %s, got %s", expectedType, typedData.Type)
	}

	// Re-marshal the data part and unmarshal to the target object
	dataBz, err := c.inner.Marshal(typedData.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	return c.inner.Unmarshal(dataBz, obj)
}

// MarshalJSON encodes an object to JSON with type information
func (c *TypedCodec) MarshalJSON(obj interface{}) ([]byte, error) {
	return c.Marshal(obj)
}

// UnmarshalJSON decodes JSON with type verification
func (c *TypedCodec) UnmarshalJSON(bz []byte, obj interface{}) error {
	return c.Unmarshal(bz, obj)
}
