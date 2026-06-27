package strategy

import (
	"fmt"
	"math"
	"strconv"

	"sentra/internal/marketdata"
	"sentra/internal/pine"
)

type PineEvaluator struct {
	cfg      pine.IRConfig
	name     string
	version  string
	symbol   string
	interval string
}

func NewPineEvaluator(name, version, symbol, interval string, cfg pine.IRConfig) *PineEvaluator {
	return &PineEvaluator{
		cfg:      cfg,
		name:     name,
		version:  version,
		symbol:   symbol,
		interval: interval,
	}
}

func (e *PineEvaluator) Evaluate(candles []marketdata.Candle) (Signal, error) {
	signals, err := e.EvaluateSeries(candles)
	if err != nil {
		return Signal{}, err
	}
	return signals[len(signals)-1], nil
}

func (e *PineEvaluator) EvaluateSeries(candles []marketdata.Candle) ([]Signal, error) {
	if len(candles) == 0 {
		return nil, fmt.Errorf("no candles to evaluate")
	}

	n := len(candles)
	opens := make([]float64, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	volumes := make([]float64, n)

	var err error
	for i := 0; i < n; i++ {
		opens[i], err = strconv.ParseFloat(candles[i].Open, 64)
		if err != nil {
			return nil, fmt.Errorf("parse open price at index %d: %w", i, err)
		}
		highs[i], err = strconv.ParseFloat(candles[i].High, 64)
		if err != nil {
			return nil, fmt.Errorf("parse high price at index %d: %w", i, err)
		}
		lows[i], err = strconv.ParseFloat(candles[i].Low, 64)
		if err != nil {
			return nil, fmt.Errorf("parse low price at index %d: %w", i, err)
		}
		closes[i], err = strconv.ParseFloat(candles[i].Close, 64)
		if err != nil {
			return nil, fmt.Errorf("parse close price at index %d: %w", i, err)
		}
		volumes[i], err = strconv.ParseFloat(candles[i].Volume, 64)
		if err != nil {
			return nil, fmt.Errorf("parse volume at index %d: %w", i, err)
		}
	}

	resolved := map[string][]float64{
		"open":   opens,
		"high":   highs,
		"low":    lows,
		"close":  closes,
		"volume": volumes,
	}

	// 3. Resolve indicator definitions topologically (dependency order)
	unresolved := make(map[string]pine.IndicatorDef)
	for k, v := range e.cfg.Indicators {
		unresolved[k] = v
	}

	for len(unresolved) > 0 {
		progress := false
		for name, def := range unresolved {
			sourceReady := true
			// For atr, source is empty/implicit
			if def.Source != "" {
				if _, exists := resolved[def.Source]; !exists {
					sourceReady = false
				}
			}

			if sourceReady {
				err := e.calculateAndResolveIndicator(name, def, resolved, candles)
				if err != nil {
					return nil, fmt.Errorf("calculate indicator %q: %w", name, err)
				}
				delete(unresolved, name)
				progress = true
			}
		}

		if !progress {
			return nil, fmt.Errorf("circular indicator dependency or unresolved indicator source in strategy")
		}
	}

	signals := make([]Signal, n)
	for index := range candles {
		side := SideHold
		reason := "no execution rules triggered"
		for _, rule := range e.cfg.Rules {
			condExpr, exists := e.cfg.Conditions[rule.Condition]
			if !exists {
				if rule.Condition == "_always_true" {
					condExpr = pine.Expression{Op: "ref", Val: "true"}
				} else {
					return nil, fmt.Errorf("undefined condition: %s", rule.Condition)
				}
			}

			val, err := evaluateExpr(condExpr, index, resolved)
			if err != nil {
				return nil, fmt.Errorf("evaluate condition %q: %w", rule.Condition, err)
			}

			triggered, ok := val.(bool)
			if ok && triggered {
				if rule.Action == "entry" {
					if rule.Direction == "long" {
						side = SideBuy
						reason = fmt.Sprintf("rule %s triggered long entry", rule.ID)
					} else if rule.Direction == "short" {
						side = SideSell
						reason = fmt.Sprintf("rule %s triggered short entry", rule.ID)
					}
				} else if rule.Action == "close" {
					if rule.Direction == "short" {
						side = SideBuy
						reason = fmt.Sprintf("rule close %s triggered cover", rule.ID)
					} else {
						side = SideSell
						reason = fmt.Sprintf("rule close %s triggered exit", rule.ID)
					}
				}
				break
			}
		}

		generatedAt := candles[index].CloseTime
		signals[index] = Signal{
			StrategyName: e.name,
			Version:      e.version,
			Symbol:       e.symbol,
			Interval:     e.interval,
			Side:         side,
			Strength:     1.0,
			Reason:       reason,
			GeneratedAt:  generatedAt,
		}
	}

	return signals, nil
}

func (e *PineEvaluator) calculateAndResolveIndicator(name string, def pine.IndicatorDef, resolved map[string][]float64, candles []marketdata.Candle) error {
	var src []float64
	if def.Source != "" {
		src = resolved[def.Source]
	}

	switch def.Type {
	case "sma":
		if len(def.Params) < 1 {
			return fmt.Errorf("sma expects 1 parameter (period)")
		}
		period := int(def.Params[0])
		resolved[name] = calculateSMA(src, period)

	case "ema":
		if len(def.Params) < 1 {
			return fmt.Errorf("ema expects 1 parameter (period)")
		}
		period := int(def.Params[0])
		resolved[name] = calculateEMA(src, period)

	case "rsi":
		if len(def.Params) < 1 {
			return fmt.Errorf("rsi expects 1 parameter (period)")
		}
		period := int(def.Params[0])
		resolved[name] = calculateRSI(src, period)

	case "atr":
		if len(def.Params) < 1 {
			return fmt.Errorf("atr expects 1 parameter (period)")
		}
		period := int(def.Params[0])
		resolved[name] = calculateATR(candles, period)

	case "macd":
		if len(def.Params) < 3 {
			return fmt.Errorf("macd expects 3 parameters (fast, slow, signal)")
		}
		fast := int(def.Params[0])
		slow := int(def.Params[1])
		signal := int(def.Params[2])
		line, sig, hist := calculateMACD(src, fast, slow, signal)
		resolved[name] = line
		resolved[name+".signal"] = sig
		resolved[name+".hist"] = hist

	case "bb":
		if len(def.Params) < 2 {
			return fmt.Errorf("bb expects 2 parameters (period, stddev)")
		}
		period := int(def.Params[0])
		stddev := def.Params[1]
		basis, upper, lower := calculateBB(src, period, stddev)
		resolved[name] = basis
		resolved[name+".upper"] = upper
		resolved[name+".lower"] = lower

	default:
		return fmt.Errorf("unsupported indicator type %q", def.Type)
	}

	return nil
}

// Indicator computations

func calculateSMA(src []float64, period int) []float64 {
	out := make([]float64, len(src))
	if period <= 0 || len(src) < period {
		return out
	}
	var sum float64
	for i := 0; i < period; i++ {
		sum += src[i]
	}
	out[period-1] = sum / float64(period)
	for i := period; i < len(src); i++ {
		sum = sum - src[i-period] + src[i]
		out[i] = sum / float64(period)
	}
	return out
}

func calculateEMA(src []float64, period int) []float64 {
	out := make([]float64, len(src))
	if period <= 0 || len(src) < period {
		return out
	}
	var sum float64
	for i := 0; i < period; i++ {
		sum += src[i]
	}
	out[period-1] = sum / float64(period)

	multiplier := 2.0 / float64(period+1)
	for i := period; i < len(src); i++ {
		out[i] = (src[i]-out[i-1])*multiplier + out[i-1]
	}
	return out
}

func calculateRSI(src []float64, period int) []float64 {
	out := make([]float64, len(src))
	if period <= 0 || len(src) < period+1 {
		return out
	}

	changes := make([]float64, len(src))
	for i := 1; i < len(src); i++ {
		changes[i] = src[i] - src[i-1]
	}

	var avgGain, avgLoss float64
	for i := 1; i <= period; i++ {
		c := changes[i]
		if c > 0 {
			avgGain += c
		} else if c < 0 {
			avgLoss += -c
		}
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	if avgLoss == 0 {
		if avgGain == 0 {
			out[period] = 50
		} else {
			out[period] = 100
		}
	} else {
		out[period] = 100 - (100 / (1 + avgGain/avgLoss))
	}

	for i := period + 1; i < len(src); i++ {
		c := changes[i]
		var gain, loss float64
		if c > 0 {
			gain = c
		} else if c < 0 {
			loss = -c
		}
		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)

		if avgLoss == 0 {
			if avgGain == 0 {
				out[i] = 50
			} else {
				out[i] = 100
			}
		} else {
			rs := avgGain / avgLoss
			out[i] = 100 - (100 / (1 + rs))
		}
	}
	return out
}

func calculateATR(candles []marketdata.Candle, period int) []float64 {
	out := make([]float64, len(candles))
	if period <= 0 || len(candles) < period+1 {
		return out
	}

	tr := make([]float64, len(candles))
	for i := 0; i < len(candles); i++ {
		h, _ := strconv.ParseFloat(candles[i].High, 64)
		l, _ := strconv.ParseFloat(candles[i].Low, 64)
		if i == 0 {
			tr[i] = h - l
			continue
		}
		prevC, _ := strconv.ParseFloat(candles[i-1].Close, 64)
		v1 := h - l
		v2 := math.Abs(h - prevC)
		v3 := math.Abs(l - prevC)
		tr[i] = math.Max(v1, math.Max(v2, v3))
	}

	var sum float64
	for i := 1; i <= period; i++ {
		sum += tr[i]
	}
	out[period] = sum / float64(period)

	for i := period + 1; i < len(candles); i++ {
		out[i] = (out[i-1]*float64(period-1) + tr[i]) / float64(period)
	}
	return out
}

func calculateMACD(src []float64, fast, slow, signal int) (line []float64, sig []float64, hist []float64) {
	fastEMA := calculateEMA(src, fast)
	slowEMA := calculateEMA(src, slow)
	line = make([]float64, len(src))
	for i := 0; i < len(src); i++ {
		line[i] = fastEMA[i] - slowEMA[i]
	}
	sig = calculateEMA(line, signal)
	hist = make([]float64, len(src))
	for i := 0; i < len(src); i++ {
		hist[i] = line[i] - sig[i]
	}
	return line, sig, hist
}

func calculateStdev(src []float64, period int) []float64 {
	out := make([]float64, len(src))
	if period <= 0 || len(src) < period {
		return out
	}
	sma := calculateSMA(src, period)
	for i := period - 1; i < len(src); i++ {
		var sumOfSquares float64
		mean := sma[i]
		for j := i - period + 1; j <= i; j++ {
			diff := src[j] - mean
			sumOfSquares += diff * diff
		}
		out[i] = math.Sqrt(sumOfSquares / float64(period))
	}
	return out
}

func calculateBB(src []float64, period int, stddev float64) (basis []float64, upper []float64, lower []float64) {
	basis = calculateSMA(src, period)
	devs := calculateStdev(src, period)
	upper = make([]float64, len(src))
	lower = make([]float64, len(src))
	for i := 0; i < len(src); i++ {
		upper[i] = basis[i] + stddev*devs[i]
		lower[i] = basis[i] - stddev*devs[i]
	}
	return basis, upper, lower
}

// Expression evaluator

func evaluateExpr(expr pine.Expression, index int, resolved map[string][]float64) (any, error) {
	if expr.Op == "ref" {
		if val, err := strconv.ParseFloat(expr.Val, 64); err == nil {
			return val, nil
		}
		if expr.Val == "true" {
			return true, nil
		}
		if expr.Val == "false" {
			return false, nil
		}
		arr, exists := resolved[expr.Val]
		if !exists {
			return nil, fmt.Errorf("undefined variable: %s", expr.Val)
		}
		if index < 0 || index >= len(arr) {
			return 0.0, nil
		}
		return arr[index], nil
	}

	if expr.Op == "not" {
		if len(expr.Args) != 1 {
			return nil, fmt.Errorf("operator 'not' expects 1 argument")
		}
		val, err := evaluateExpr(expr.Args[0], index, resolved)
		if err != nil {
			return nil, err
		}
		b, ok := val.(bool)
		if !ok {
			return nil, fmt.Errorf("operator 'not' expects boolean operand")
		}
		return !b, nil
	}

	if len(expr.Args) != 2 {
		return nil, fmt.Errorf("operator %q expects 2 arguments", expr.Op)
	}

	leftVal, err := evaluateExpr(expr.Args[0], index, resolved)
	if err != nil {
		return nil, err
	}
	rightVal, err := evaluateExpr(expr.Args[1], index, resolved)
	if err != nil {
		return nil, err
	}

	switch expr.Op {
	case ">", "<", ">=", "<=", "==", "!=":
		leftFloat, ok1 := leftVal.(float64)
		rightFloat, ok2 := rightVal.(float64)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("comparison operator %q expects numeric operands", expr.Op)
		}
		switch expr.Op {
		case ">":
			return leftFloat > rightFloat, nil
		case "<":
			return leftFloat < rightFloat, nil
		case ">=":
			return leftFloat >= rightFloat, nil
		case "<=":
			return leftFloat <= rightFloat, nil
		case "==":
			return leftFloat == rightFloat, nil
		case "!=":
			return leftFloat != rightFloat, nil
		}

	case "and", "or":
		leftBool, ok1 := leftVal.(bool)
		rightBool, ok2 := rightVal.(bool)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("logical operator %q expects boolean operands", expr.Op)
		}
		if expr.Op == "and" {
			return leftBool && rightBool, nil
		}
		return leftBool || rightBool, nil

	case "crossover", "crossunder":
		if index < 1 {
			return false, nil
		}
		if expr.Args[0].Op != "ref" || expr.Args[1].Op != "ref" {
			return nil, fmt.Errorf("crossover/crossunder arguments must be variable references")
		}
		nameA := expr.Args[0].Val
		nameB := expr.Args[1].Val

		arrA, existsA := resolved[nameA]
		arrB, existsB := resolved[nameB]

		if !existsA {
			if val, err := strconv.ParseFloat(nameA, 64); err == nil {
				arrA = make([]float64, index+1)
				for j := range arrA {
					arrA[j] = val
				}
			} else {
				return nil, fmt.Errorf("undefined variable: %s", nameA)
			}
		}
		if !existsB {
			if val, err := strconv.ParseFloat(nameB, 64); err == nil {
				arrB = make([]float64, index+1)
				for j := range arrB {
					arrB[j] = val
				}
			} else {
				return nil, fmt.Errorf("undefined variable: %s", nameB)
			}
		}

		prevA := arrA[index-1]
		prevB := arrB[index-1]
		currA := arrA[index]
		currB := arrB[index]

		gap := 0.0001 * currB

		if expr.Op == "crossover" {
			return prevA < prevB && currA > currB && (currA-currB) > gap, nil
		} else {
			return prevA > prevB && currA < currB && (currB-currA) > gap, nil
		}
	}

	return nil, fmt.Errorf("unsupported operator: %s", expr.Op)
}
