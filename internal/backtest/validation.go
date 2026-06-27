package backtest

import (
	"fmt"
	"strconv"
	"time"

	"sentra/internal/marketdata"
)

type CandleSeriesDiagnostics struct {
	Valid            bool   `json:"valid"`
	Count            int    `json:"count"`
	GapCount         int    `json:"gap_count"`
	DuplicateCount   int    `json:"duplicate_count"`
	OutOfOrderCount  int    `json:"out_of_order_count"`
	UnclosedCount    int    `json:"unclosed_count"`
	InvalidOHLCCount int    `json:"invalid_ohlc_count"`
	SymbolMismatch   int    `json:"symbol_mismatch"`
	IntervalMismatch int    `json:"interval_mismatch"`
	Reason           string `json:"reason"`
}

type CandleSeriesError struct {
	Diagnostics CandleSeriesDiagnostics
}

func (e CandleSeriesError) Error() string {
	if e.Diagnostics.Reason == "" {
		return "invalid candle series"
	}
	return "invalid candle series: " + e.Diagnostics.Reason
}

func ValidateCandleSeries(symbol string, interval string, candles []marketdata.Candle) CandleSeriesDiagnostics {
	diagnostics := CandleSeriesDiagnostics{
		Valid: true,
		Count: len(candles),
	}
	if len(candles) == 0 {
		diagnostics.Valid = false
		diagnostics.Reason = "no candles"
		return diagnostics
	}

	step, err := marketdata.IntervalDuration(interval)
	if err != nil {
		diagnostics.Valid = false
		diagnostics.Reason = err.Error()
		return diagnostics
	}

	seen := make(map[time.Time]struct{}, len(candles))
	for index, candle := range candles {
		if candle.Symbol != "" && candle.Symbol != symbol {
			diagnostics.SymbolMismatch++
		}
		if candle.Interval != "" && candle.Interval != interval {
			diagnostics.IntervalMismatch++
		}
		if !candle.IsClosed {
			diagnostics.UnclosedCount++
		}
		if !validOHLC(candle) {
			diagnostics.InvalidOHLCCount++
		}
		openTime := candle.OpenTime.UTC()
		if _, exists := seen[openTime]; exists {
			diagnostics.DuplicateCount++
		}
		seen[openTime] = struct{}{}

		if index == 0 {
			continue
		}
		previous := candles[index-1].OpenTime.UTC()
		if !openTime.After(previous) {
			diagnostics.OutOfOrderCount++
			continue
		}
		if openTime.Sub(previous) != step {
			diagnostics.GapCount++
		}
	}

	if diagnostics.GapCount > 0 ||
		diagnostics.DuplicateCount > 0 ||
		diagnostics.OutOfOrderCount > 0 ||
		diagnostics.UnclosedCount > 0 ||
		diagnostics.InvalidOHLCCount > 0 ||
		diagnostics.SymbolMismatch > 0 ||
		diagnostics.IntervalMismatch > 0 {
		diagnostics.Valid = false
		diagnostics.Reason = "candle series failed validation"
	}
	return diagnostics
}

func validOHLC(candle marketdata.Candle) bool {
	closeValue, err := parsePositivePrice(candle.Close, "close")
	if err != nil {
		return false
	}
	openValue := closeValue
	if candle.Open != "" {
		value, err := parsePositivePrice(candle.Open, "open")
		if err != nil {
			return false
		}
		openValue = value
	}
	highValue := maxFloat(openValue, closeValue)
	if candle.High != "" {
		value, err := parsePositivePrice(candle.High, "high")
		if err != nil {
			return false
		}
		highValue = value
	}
	lowValue := minFloat(openValue, closeValue)
	if candle.Low != "" {
		value, err := parsePositivePrice(candle.Low, "low")
		if err != nil {
			return false
		}
		lowValue = value
	}
	return highValue >= openValue && highValue >= closeValue && lowValue <= openValue && lowValue <= closeValue && lowValue <= highValue
}

func parsePositivePrice(value string, field string) (float64, error) {
	if value == "" {
		return 0, fmt.Errorf("%s is required", field)
	}
	price, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}
	if price <= 0 {
		return 0, fmt.Errorf("%s must be positive", field)
	}
	return price, nil
}

func maxFloat(left float64, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func minFloat(left float64, right float64) float64 {
	if left < right {
		return left
	}
	return right
}

// SanityWarning describes a configuration or result anomaly that is likely a bug.
type SanityWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SanityCheckRequest inspects a backtest request for obviously dangerous
// configurations that are known to produce catastrophic results. These checks
// encode lessons from past incidents where the combination of all_in sizing,
// zero cooldown, and high-frequency intervals destroyed 99%+ of capital through
// fee compounding. This function exists to make those failures impossible to
// reproduce silently.
func SanityCheckRequest(request Request) []SanityWarning {
	var warnings []SanityWarning

	// All-in sizing with non-zero fees will compound losses on every round trip.
	if request.PositionSizingMode == PositionSizingAllIn && request.FeeRate > 0 {
		warnings = append(warnings, SanityWarning{
			Code:    "dangerous_position_sizing",
			Message: "all_in position sizing with fees causes multiplicative capital drain; consider percent_equity",
		})
	}

	// Zero cooldown on sub-hourly intervals invites signal thrashing.
	if request.CooldownBars <= 0 {
		warnings = append(warnings, SanityWarning{
			Code:    "no_cooldown",
			Message: "cooldown_bars is zero; the engine can re-enter immediately after exit, causing overtrading",
		})
	}

	// Zero minimum holding on sub-hourly intervals invites churn.
	if request.MinHoldingBars <= 0 {
		warnings = append(warnings, SanityWarning{
			Code:    "no_min_holding",
			Message: "min_holding_bars is zero; positions can be closed on the next candle, amplifying fee drag",
		})
	}

	// Missing fee rate produces unrealistically optimistic results.
	if request.FeeRate <= 0 {
		warnings = append(warnings, SanityWarning{
			Code:    "no_fees",
			Message: "fee_rate is zero; results will not reflect real trading costs",
		})
	}

	return warnings
}

// SanityCheckResult inspects a completed backtest run for anomalies that
// indicate engine bugs or data quality issues. A return below -90% with
// hundreds of trades is the signature of the fee-compounding bug.
func SanityCheckResult(run Run) []SanityWarning {
	var warnings []SanityWarning

	if run.ReturnPercent < -90 && run.TotalTrades > 100 {
		warnings = append(warnings, SanityWarning{
			Code:    "catastrophic_loss",
			Message: "return is below -90% with 100+ trades; this is almost certainly a fee-compounding bug, not a strategy failure",
		})
	}

	if run.TotalTrades > 0 && run.EndingBalance > 0 {
		avgTradeSize := run.StartingBalance / float64(run.TotalTrades)
		if avgTradeSize < 0.01 {
			warnings = append(warnings, SanityWarning{
				Code:    "dust_trades",
				Message: "average trade size is below $0.01; capital was eroded to dust by excessive round trips",
			})
		}
	}

	return warnings
}
