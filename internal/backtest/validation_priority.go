package backtest

func validationRank(status string) int {
	switch status {
	case "candidate":
		return 0
	case "weak_profit_factor", "insufficient_sample", "negative_average_trade":
		return 10
	case "underperforms_benchmark", "underperforms_walk_forward", "unstable_walk_forward":
		return 20
	case "low_bull_market_capture":
		return 30
	case "cost_drag", "overtrading":
		return 40
	case "unsafe_execution_timing", "high_drawdown":
		return 50
	case "":
		return 90
	default:
		return 60
	}
}

func validationRankLess(left string, right string) bool {
	return validationRank(left) < validationRank(right)
}

func maxValidationRank(statuses ...string) int {
	maxRank := 0
	hasStatus := false
	for _, status := range statuses {
		if status == "" {
			continue
		}
		rank := validationRank(status)
		if !hasStatus || rank > maxRank {
			maxRank = rank
			hasStatus = true
		}
	}
	if !hasStatus {
		return validationRank("")
	}
	return maxRank
}
