package types

import (
	"fmt"
	"strconv"
	"strings"
)

// PruningStrategy defines the pruning strategy
type PruningStrategy uint8

const (
	// PruningDefault defines a pruning strategy where the last 362,880 heights are kept
	// where to-be pruned heights are pruned at every 10th height.
	PruningDefault PruningStrategy = iota

	// PruningEverything defines a pruning strategy where all committed heights are deleted,
	// storing only the current height and where to-be pruned heights are pruned at every 10th height.
	PruningEverything

	// PruningNothing defines a pruning strategy where all heights are kept on disk.
	PruningNothing

	// PruningCustom defines a pruning strategy where the user defines the intervals
	PruningCustom
)

const (
	// DefaultKeepRecent is the default number of recent heights to keep
	DefaultKeepRecent uint64 = 362880
	// DefaultKeepEvery is the default interval for keeping heights
	DefaultKeepEvery uint64 = 0
	// DefaultInterval is the default pruning interval
	DefaultInterval uint64 = 10
)

// PruningOptions defines the pruning configuration
type PruningOptions struct {
	// KeepRecent defines how many recent heights to keep on disk
	KeepRecent uint64

	// KeepEvery defines the offset heights to keep on disk after KeepRecent
	KeepEvery uint64

	// Interval defines when the pruned heights are removed from disk
	Interval uint64

	// Strategy defines the pruning strategy
	Strategy PruningStrategy
}

// NewPruningOptions creates a new PruningOptions for the given strategy
func NewPruningOptions(strategy PruningStrategy) PruningOptions {
	switch strategy {
	case PruningDefault:
		return PruningOptions{
			KeepRecent: DefaultKeepRecent,
			KeepEvery:  DefaultKeepEvery,
			Interval:   DefaultInterval,
			Strategy:   PruningDefault,
		}
	case PruningEverything:
		return PruningOptions{
			KeepRecent: 2,
			KeepEvery:  0,
			Interval:   DefaultInterval,
			Strategy:   PruningEverything,
		}
	case PruningNothing:
		return PruningOptions{
			KeepRecent: 0,
			KeepEvery:  1,
			Interval:   0,
			Strategy:   PruningNothing,
		}
	default:
		return PruningOptions{
			Strategy: PruningCustom,
		}
	}
}

// NewCustomPruningOptions creates a new custom PruningOptions
func NewCustomPruningOptions(keepRecent, keepEvery, interval uint64) PruningOptions {
	return PruningOptions{
		KeepRecent: keepRecent,
		KeepEvery:  keepEvery,
		Interval:   interval,
		Strategy:   PruningCustom,
	}
}

// Validate validates the pruning options
func (po PruningOptions) Validate() error {
	if po.Strategy == PruningCustom {
		if po.KeepRecent < 2 {
			return fmt.Errorf("keep recent must be greater than 1, got %d", po.KeepRecent)
		}
		if po.Interval == 0 {
			return fmt.Errorf("pruning interval must be greater than 0")
		}
	}
	return nil
}

// GetStrategy returns the pruning strategy
func (po PruningOptions) GetStrategy() PruningStrategy {
	return po.Strategy
}

// String returns a string representation of the pruning options
func (po PruningOptions) String() string {
	switch po.Strategy {
	case PruningDefault:
		return "default"
	case PruningEverything:
		return "everything"
	case PruningNothing:
		return "nothing"
	case PruningCustom:
		return fmt.Sprintf("custom(keep_recent=%d,keep_every=%d,interval=%d)", 
			po.KeepRecent, po.KeepEvery, po.Interval)
	default:
		return "unknown"
	}
}

// ParsePruningOptionsFromString parses pruning options from string
func ParsePruningOptionsFromString(s string) (PruningOptions, error) {
	switch s {
	case "default":
		return NewPruningOptions(PruningDefault), nil
	case "everything":
		return NewPruningOptions(PruningEverything), nil
	case "nothing":
		return NewPruningOptions(PruningNothing), nil
	default:
		if strings.HasPrefix(s, "custom") {
			return parseCustomPruning(s)
		}
		return PruningOptions{}, fmt.Errorf("unknown pruning strategy: %s", s)
	}
}

func parseCustomPruning(s string) (PruningOptions, error) {
	// Parse custom(keep_recent=X,keep_every=Y,interval=Z)
	if !strings.HasPrefix(s, "custom(") || !strings.HasSuffix(s, ")") {
		return PruningOptions{}, fmt.Errorf("invalid custom pruning format: %s", s)
	}
	
	inner := s[7 : len(s)-1] // Remove "custom(" and ")"
	parts := strings.Split(inner, ",")
	
	if len(parts) != 3 {
		return PruningOptions{}, fmt.Errorf("custom pruning must have 3 parameters: %s", s)
	}
	
	var keepRecent, keepEvery, interval uint64
	var err error
	
	for _, part := range parts {
		kv := strings.Split(strings.TrimSpace(part), "=")
		if len(kv) != 2 {
			return PruningOptions{}, fmt.Errorf("invalid parameter format: %s", part)
		}
		
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])
		
		switch key {
		case "keep_recent":
			keepRecent, err = strconv.ParseUint(value, 10, 64)
		case "keep_every":
			keepEvery, err = strconv.ParseUint(value, 10, 64)
		case "interval":
			interval, err = strconv.ParseUint(value, 10, 64)
		default:
			return PruningOptions{}, fmt.Errorf("unknown parameter: %s", key)
		}
		
		if err != nil {
			return PruningOptions{}, fmt.Errorf("invalid value for %s: %s", key, value)
		}
	}
	
	return NewCustomPruningOptions(keepRecent, keepEvery, interval), nil
}

// ShouldPrune returns whether the given height should be pruned
func (po PruningOptions) ShouldPrune(currentHeight, version int64) bool {
	if po.Strategy == PruningNothing {
		return false
	}
	
	if version <= 0 {
		return false
	}
	
	if po.Strategy == PruningEverything {
		// Keep only the last two versions
		return currentHeight-version > int64(po.KeepRecent)
	}
	
	// For default and custom strategies
	if currentHeight-version <= int64(po.KeepRecent) {
		return false
	}
	
	if po.KeepEvery > 0 && version%int64(po.KeepEvery) == 0 {
		return false
	}
	
	return true
}