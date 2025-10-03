package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/signal/types"
)

// GetQueryCmd returns the cli query commands for the signal module
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       keepertypes.ValidateCmd,
	}

	cmd.AddCommand(
		CmdQuerySignal(),
		CmdQuerySignals(),
		CmdQueryValidatorSignals(),
		CmdQuerySignalScore(),
		CmdQueryValidatorStats(),
		CmdQueryLeaderboard(),
		CmdQueryWorkType(),
		CmdQueryWorkTypes(),
		CmdQuerySignalStats(),
		CmdQueryParams(),
		CmdQueryVerifications(),
		CmdQueryRecentSignals(),
		CmdQueryTopValidators(),
		CmdQuerySignalsByStatus(),
		CmdQuerySignalsByWorkType(),
		CmdQueryValidatorRank(),
		CmdQueryScoreDistribution(),
	)

	return cmd
}

// CmdQuerySignal queries a single signal by ID
func CmdQuerySignal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signal [signal-id]",
		Short: "Query a signal by ID",
		Long: `Query a specific signal by its ID.

Example:
$ pulsarcli query signal signal "signal_abc123"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			signalID := args[0]

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QuerySignalRequest{
				SignalID: signalID,
			}

			res, err := queryClient.Signal(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	keepertypes.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQuerySignals queries multiple signals with filters
func CmdQuerySignals() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signals",
		Short: "Query signals with optional filters",
		Long: `Query signals with optional filters for creator, validator, type, status, etc.

Example:
$ pulsarcli query signal signals --creator cosmos1... --limit 10
$ pulsarcli query signal signals --validator cosmos1validator... --status validated
$ pulsarcli query signal signals --work-type llm_inference --min-score 100`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			// Parse filter flags
			filter, err := parseSignalFilterFromFlags(cmd)
			if err != nil {
				return fmt.Errorf("failed to parse filters: %w", err)
			}

			limit, _ := cmd.Flags().GetInt("limit")
			offset, _ := cmd.Flags().GetInt("offset")

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QuerySignalsRequest{
				Filter: filter,
				Limit:  limit,
				Offset: offset,
			}

			res, err := queryClient.Signals(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	addSignalFilterFlags(cmd)
	cmd.Flags().Int("limit", 50, "Limit the number of results")
	cmd.Flags().Int("offset", 0, "Offset for pagination")
	keepertypes.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryValidatorSignals queries signals for a specific validator
func CmdQueryValidatorSignals() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validator-signals [validator-address]",
		Short: "Query signals for a specific validator",
		Long: `Query all signals associated with a specific validator.

Example:
$ pulsarcli query signal validator-signals cosmos1validator... --limit 20`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			validatorAddr := args[0]
			limit, _ := cmd.Flags().GetInt("limit")
			offset, _ := cmd.Flags().GetInt("offset")

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryValidatorSignalsRequest{
				ValidatorAddress: validatorAddr,
				Limit:           limit,
				Offset:          offset,
			}

			res, err := queryClient.ValidatorSignals(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Int("limit", 50, "Limit the number of results")
	cmd.Flags().Int("offset", 0, "Offset for pagination")
	keepertypes.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQuerySignalScore queries the score for a specific signal
func CmdQuerySignalScore() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signal-score [signal-id]",
		Short: "Query the score for a specific signal",
		Long: `Query the calculated score and scoring details for a specific signal.

Example:
$ pulsarcli query signal signal-score "signal_abc123"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			signalID := args[0]

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QuerySignalScoreRequest{
				SignalID: signalID,
			}

			res, err := queryClient.SignalScore(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	keepertypes.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryValidatorStats queries statistics for a validator
func CmdQueryValidatorStats() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validator-stats [validator-address]",
		Short: "Query statistics for a validator",
		Long: `Query comprehensive statistics for a validator including total signals, scores, and rank.

Example:
$ pulsarcli query signal validator-stats cosmos1validator...`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			validatorAddr := args[0]

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryValidatorStatsRequest{
				ValidatorAddress: validatorAddr,
			}

			res, err := queryClient.ValidatorStats(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	keepertypes.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryLeaderboard queries the validator leaderboard
func CmdQueryLeaderboard() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "leaderboard",
		Short: "Query the validator leaderboard",
		Long: `Query the validator leaderboard ranked by signal scores.

Example:
$ pulsarcli query signal leaderboard --limit 10`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetInt("limit")

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryLeaderboardRequest{
				Limit: limit,
			}

			res, err := queryClient.Leaderboard(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Int("limit", 50, "Limit the number of results")
	keepertypes.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryWorkType queries a specific work type
func CmdQueryWorkType() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "work-type [work-type-id]",
		Short: "Query a specific AI work type",
		Long: `Query the configuration and details of a specific AI work type.

Example:
$ pulsarcli query signal work-type "llm_inference"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			workTypeID := args[0]

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryWorkTypeRequest{
				WorkTypeID: workTypeID,
			}

			res, err := queryClient.WorkType(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	keepertypes.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryWorkTypes queries all work types
func CmdQueryWorkTypes() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "work-types",
		Short: "Query all AI work types",
		Long: `Query all registered AI work types, optionally filtering for enabled only.

Example:
$ pulsarcli query signal work-types
$ pulsarcli query signal work-types --enabled-only`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			enabledOnly, _ := cmd.Flags().GetBool("enabled-only")

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryWorkTypesRequest{
				EnabledOnly: enabledOnly,
			}

			res, err := queryClient.WorkTypes(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Bool("enabled-only", false, "Show only enabled work types")
	keepertypes.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQuerySignalStats queries overall signal statistics
func CmdQuerySignalStats() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signal-stats",
		Short: "Query overall signal statistics",
		Long: `Query overall statistics about signals including counts, validation rates, and score distributions.

Example:
$ pulsarcli query signal signal-stats`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QuerySignalStatsRequest{}

			res, err := queryClient.SignalStats(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	keepertypes.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryParams queries module parameters
func CmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query signal module parameters",
		Long: `Query the current parameters of the signal module.

Example:
$ pulsarcli query signal params`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryParamsRequest{}

			res, err := queryClient.Params(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	keepertypes.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryVerifications queries work verifications for a signal
func CmdQueryVerifications() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verifications [signal-id]",
		Short: "Query work verifications for a signal",
		Long: `Query all work verifications submitted for a specific signal.

Example:
$ pulsarcli query signal verifications "signal_abc123"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			signalID := args[0]

			// This would need to be implemented in the query server
			// For now, we'll use a direct keeper query approach
			fmt.Printf("Querying verifications for signal: %s\n", signalID)
			fmt.Println("This feature requires implementation of verification query endpoints.")

			return nil
		},
	}

	keepertypes.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryRecentSignals queries recent signals
func CmdQueryRecentSignals() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recent-signals",
		Short: "Query recent signals",
		Long: `Query the most recently submitted signals.

Example:
$ pulsarcli query signal recent-signals --limit 20`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetInt("limit")

			// Use signals query with time-based filter
			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QuerySignalsRequest{
				Filter: types.SignalFilter{
					// Would filter by recent time range
				},
				Limit:  limit,
				Offset: 0,
			}

			res, err := queryClient.Signals(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Int("limit", 20, "Limit the number of results")
	keepertypes.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryTopValidators queries top validators by signal score
func CmdQueryTopValidators() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "top-validators",
		Short: "Query top validators by signal score",
		Long: `Query the top validators ranked by their signal scores.

Example:
$ pulsarcli query signal top-validators --limit 10`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetInt("limit")

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryLeaderboardRequest{
				Limit: limit,
			}

			res, err := queryClient.Leaderboard(cmd.Context(), req)
			if err != nil {
				return err
			}

			// Format as top validators list
			fmt.Printf("Top %d validators by signal score:\n", limit)
			for i, entry := range res.Leaderboard.Entries {
				fmt.Printf("%d. %s - Score: %d (%.2f%% of total)\n",
					i+1, entry.ValidatorAddress, entry.Score, entry.PercentOfTotal)
			}

			return nil
		},
	}

	cmd.Flags().Int("limit", 10, "Limit the number of results")
	keepertypes.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQuerySignalsByStatus queries signals by status
func CmdQuerySignalsByStatus() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signals-by-status [status]",
		Short: "Query signals by their status",
		Long: `Query signals filtered by their processing status.

Status can be: pending, validated, rejected, expired

Example:
$ pulsarcli query signal signals-by-status validated --limit 10`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			statusStr := args[0]
			status, err := parseSignalStatus(statusStr)
			if err != nil {
				return fmt.Errorf("invalid status: %w", err)
			}

			limit, _ := cmd.Flags().GetInt("limit")
			offset, _ := cmd.Flags().GetInt("offset")

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QuerySignalsRequest{
				Filter: types.SignalFilter{
					Status: status,
				},
				Limit:  limit,
				Offset: offset,
			}

			res, err := queryClient.Signals(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Int("limit", 50, "Limit the number of results")
	cmd.Flags().Int("offset", 0, "Offset for pagination")
	keepertypes.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQuerySignalsByWorkType queries signals by work type
func CmdQuerySignalsByWorkType() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signals-by-work-type [work-type]",
		Short: "Query signals by work type",
		Long: `Query signals filtered by their AI work type.

Example:
$ pulsarcli query signal signals-by-work-type llm_inference --limit 10`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			workType := args[0]
			limit, _ := cmd.Flags().GetInt("limit")
			offset, _ := cmd.Flags().GetInt("offset")

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QuerySignalsRequest{
				Filter: types.SignalFilter{
					Type: workType,
				},
				Limit:  limit,
				Offset: offset,
			}

			res, err := queryClient.Signals(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Int("limit", 50, "Limit the number of results")
	cmd.Flags().Int("offset", 0, "Offset for pagination")
	keepertypes.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryValidatorRank queries the rank of a specific validator
func CmdQueryValidatorRank() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validator-rank [validator-address]",
		Short: "Query the rank of a specific validator",
		Long: `Query the current rank of a validator in the signal leaderboard.

Example:
$ pulsarcli query signal validator-rank cosmos1validator...`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			validatorAddr := args[0]

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryValidatorStatsRequest{
				ValidatorAddress: validatorAddr,
			}

			res, err := queryClient.ValidatorStats(cmd.Context(), req)
			if err != nil {
				return err
			}

			fmt.Printf("Validator %s rank: %d\n", validatorAddr, res.Rank)
			fmt.Printf("Signal score: %d\n", res.Stats.EffectiveScore)
			fmt.Printf("Total signals: %d\n", res.Stats.TotalSignals)

			return nil
		},
	}

	keepertypes.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryScoreDistribution queries score distribution analysis
func CmdQueryScoreDistribution() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "score-distribution",
		Short: "Query signal score distribution analysis",
		Long: `Query statistical analysis of signal score distribution.

Example:
$ pulsarcli query signal score-distribution`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := keepertypes.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QuerySignalStatsRequest{}

			res, err := queryClient.SignalStats(cmd.Context(), req)
			if err != nil {
				return err
			}

			// Extract and display score distribution from stats
			if scoreDistribution, ok := res.Stats["score_distribution"]; ok {
				fmt.Println("Signal Score Distribution:")
				fmt.Printf("Raw data: %+v\n", scoreDistribution)
			} else {
				fmt.Println("Score distribution data not available")
			}

			return nil
		},
	}

	keepertypes.AddQueryFlagsToCmd(cmd)
	return cmd
}

// Helper functions

// addSignalFilterFlags adds filter flags for signal queries
func addSignalFilterFlags(cmd *cobra.Command) {
	cmd.Flags().String("creator", "", "Filter by signal creator address")
	cmd.Flags().String("validator", "", "Filter by validator address")
	cmd.Flags().String("type", "", "Filter by signal type")
	cmd.Flags().String("work-type", "", "Filter by work type")
	cmd.Flags().String("status", "", "Filter by signal status (pending, validated, rejected, expired)")
	cmd.Flags().Uint64("min-score", 0, "Filter by minimum score")
	cmd.Flags().Uint64("max-score", 0, "Filter by maximum score")
	cmd.Flags().Int64("start-time", 0, "Filter by start time (unix timestamp)")
	cmd.Flags().Int64("end-time", 0, "Filter by end time (unix timestamp)")
}

// parseSignalFilterFromFlags parses signal filter from command flags
func parseSignalFilterFromFlags(cmd *cobra.Command) (types.SignalFilter, error) {
	filter := types.SignalFilter{}

	if creator, _ := cmd.Flags().GetString("creator"); creator != "" {
		filter.Creator = creator
	}

	if validator, _ := cmd.Flags().GetString("validator"); validator != "" {
		filter.ValidatorAddress = validator
	}

	if signalType, _ := cmd.Flags().GetString("type"); signalType != "" {
		filter.Type = signalType
	}

	if statusStr, _ := cmd.Flags().GetString("status"); statusStr != "" {
		status, err := parseSignalStatus(statusStr)
		if err != nil {
			return filter, err
		}
		filter.Status = status
	}

	if minScore, _ := cmd.Flags().GetUint64("min-score"); minScore > 0 {
		filter.MinScore = minScore
	}

	if maxScore, _ := cmd.Flags().GetUint64("max-score"); maxScore > 0 {
		filter.MaxScore = maxScore
	}

	if startTime, _ := cmd.Flags().GetInt64("start-time"); startTime > 0 {
		filter.StartTime = startTime
	}

	if endTime, _ := cmd.Flags().GetInt64("end-time"); endTime > 0 {
		filter.EndTime = endTime
	}

	return filter, nil
}

// parseSignalStatus parses signal status from string
func parseSignalStatus(statusStr string) (types.SignalStatus, error) {
	switch strings.ToLower(statusStr) {
	case "pending":
		return types.SignalStatusPending, nil
	case "validated":
		return types.SignalStatusValidated, nil
	case "rejected":
		return types.SignalStatusRejected, nil
	case "expired":
		return types.SignalStatusExpired, nil
	default:
		return 0, fmt.Errorf("unknown status: %s", statusStr)
	}
}