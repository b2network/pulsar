package types

import (
	"fmt"
)

// GenesisState defines the signal module's genesis state
type GenesisState struct {
	Params        Params                          `json:"params"`
	Signals       []Signal                        `json:"signals"`
	SignalScores  []SignalScore                   `json:"signal_scores"`
	WorkTypes     []AIWorkType                    `json:"work_types"`
	ValidatorStats map[string]*ValidatorSignalStats `json:"validator_stats"`
	WorkRegistry  *AIWorkRegistry                  `json:"work_registry"`
	SignalSequence uint64                          `json:"signal_sequence"`
}

// NewGenesisState creates a new genesis state
func NewGenesisState(params Params) *GenesisState {
	return &GenesisState{
		Params:         params,
		Signals:        []Signal{},
		SignalScores:   []SignalScore{},
		WorkTypes:      DefaultAIWorkTypes(),
		ValidatorStats: make(map[string]*ValidatorSignalStats),
		WorkRegistry:   NewAIWorkRegistry(),
		SignalSequence: 0,
	}
}

// DefaultGenesisState returns the default genesis state
func DefaultGenesisState() *GenesisState {
	return NewGenesisState(DefaultParams())
}

// ValidateGenesis validates the genesis state
func ValidateGenesis(data *GenesisState) error {
	// Validate params
	if err := data.Params.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	// Validate signals
	signalIDs := make(map[string]bool)
	for i, signal := range data.Signals {
		if err := signal.ValidateBasic(); err != nil {
			return fmt.Errorf("invalid signal at index %d: %w", i, err)
		}
		if signalIDs[signal.ID] {
			return fmt.Errorf("duplicate signal ID: %s", signal.ID)
		}
		signalIDs[signal.ID] = true
	}

	// Validate signal scores
	for i, score := range data.SignalScores {
		if score.SignalID == "" {
			return fmt.Errorf("invalid signal score at index %d: empty signal ID", i)
		}
		if score.FinalScore == 0 && score.BaseScore == 0 {
			return fmt.Errorf("invalid signal score at index %d: zero scores", i)
		}
	}

	// Validate work types
	workTypeIDs := make(map[string]bool)
	for i, workType := range data.WorkTypes {
		if err := workType.ValidateBasic(); err != nil {
			return fmt.Errorf("invalid work type at index %d: %w", i, err)
		}
		if workTypeIDs[workType.ID] {
			return fmt.Errorf("duplicate work type ID: %s", workType.ID)
		}
		workTypeIDs[workType.ID] = true
	}

	// Validate validator stats
	for addr, stats := range data.ValidatorStats {
		if addr == "" {
			return fmt.Errorf("empty validator address in stats")
		}
		if stats == nil {
			return fmt.Errorf("nil stats for validator %s", addr)
		}
		if stats.ValidatorAddress != addr {
			return fmt.Errorf("mismatched validator address: %s != %s", addr, stats.ValidatorAddress)
		}
	}

	// Validate work registry
	if data.WorkRegistry != nil {
		for id, workType := range data.WorkRegistry.WorkTypes {
			if err := workType.ValidateBasic(); err != nil {
				return fmt.Errorf("invalid work type %s in registry: %w", id, err)
			}
		}
	}

	return nil
}

// GetSignalByID returns a signal by ID from genesis state
func (gs *GenesisState) GetSignalByID(id string) (*Signal, bool) {
	for _, signal := range gs.Signals {
		if signal.ID == id {
			return &signal, true
		}
	}
	return nil, false
}

// GetWorkTypeByID returns a work type by ID from genesis state
func (gs *GenesisState) GetWorkTypeByID(id string) (*AIWorkType, bool) {
	for _, workType := range gs.WorkTypes {
		if workType.ID == id {
			return &workType, true
		}
	}
	return nil, false
}

// AddSignal adds a signal to the genesis state
func (gs *GenesisState) AddSignal(signal Signal) error {
	if err := signal.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid signal: %w", err)
	}

	// Check for duplicates
	for _, existing := range gs.Signals {
		if existing.ID == signal.ID {
			return fmt.Errorf("signal with ID %s already exists", signal.ID)
		}
	}

	gs.Signals = append(gs.Signals, signal)
	return nil
}

// AddWorkType adds a work type to the genesis state
func (gs *GenesisState) AddWorkType(workType AIWorkType) error {
	if err := workType.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid work type: %w", err)
	}

	// Check for duplicates
	for _, existing := range gs.WorkTypes {
		if existing.ID == workType.ID {
			return fmt.Errorf("work type with ID %s already exists", workType.ID)
		}
	}

	gs.WorkTypes = append(gs.WorkTypes, workType)

	// Also add to registry if it exists
	if gs.WorkRegistry != nil {
		gs.WorkRegistry.WorkTypes[workType.ID] = workType
	}

	return nil
}

// InitializeValidatorStats initializes validator stats if not present
func (gs *GenesisState) InitializeValidatorStats(validatorAddr string) {
	if gs.ValidatorStats == nil {
		gs.ValidatorStats = make(map[string]*ValidatorSignalStats)
	}

	if _, exists := gs.ValidatorStats[validatorAddr]; !exists {
		gs.ValidatorStats[validatorAddr] = &ValidatorSignalStats{
			ValidatorAddress: validatorAddr,
			TotalSignals:     0,
			ValidSignals:     0,
			RejectedSignals:  0,
			TotalScore:       0,
			AverageScore:     0,
			LastSignalTime:   0,
			ActiveSignals:    []string{},
			RecentScores:     []uint64{},
			ScoreDecayRate:   gs.Params.ScoreDecayRate,
			EffectiveScore:   0,
			Rank:             0,
		}
	}
}