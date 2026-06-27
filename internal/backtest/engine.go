package backtest

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"sentra/internal/indicator"
	"sentra/internal/marketdata"
	"sentra/internal/strategy"
)

type Engine struct{}

type seriesEvaluator interface {
	EvaluateSeries(candles []marketdata.Candle) ([]strategy.Signal, error)
}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Run(request Request, candles []marketdata.Candle) (Run, []Trade, error) {
	request = request.Normalize()
	if err := request.Validate(); err != nil {
		return Run{}, nil, err
	}
	required := request.RequiredCandles()
	if len(candles) < required {
		return Run{}, nil, fmt.Errorf("insufficient candles: need %d, got %d", required, len(candles))
	}
	diagnostics := ValidateCandleSeries(request.Symbol, request.Interval, candles)
	if !diagnostics.Valid {
		return Run{}, nil, CandleSeriesError{Diagnostics: diagnostics}
	}

	quoteBalance := request.StartingBalance
	baseBalance := 0.0
	entryQuote := 0.0
	entryReferencePrice := 0.0
	peakEquity := request.StartingBalance
	maxDrawdown := 0.0
	grossProfitLoss := 0.0
	totalFees := 0.0
	estimatedSlippageCost := 0.0
	wins := 0
	buyCount := 0
	sellCount := 0
	completedTrades := 0
	bestTrade := 0.0
	worstTrade := 0.0
	winTotal := 0.0
	lossTotal := 0.0
	losses := 0
	roundTrips := []RoundTrip{}
	equityCurve := []EquityPoint{}
	trades := []Trade{}
	var openTrade *Trade
	entryIndex := -1
	lastTradeIndex := -1
	stopLossPrice := 0.0
	takeProfitPrice := 0.0

	evaluator, err := strategy.NewEvaluatorFromSettings(request.StrategySettings())
	if err != nil {
		return Run{}, nil, err
	}
	var signalSeries []strategy.Signal
	if evaluator, ok := evaluator.(seriesEvaluator); ok {
		signalSeries, err = evaluator.EvaluateSeries(candles)
		if err != nil {
			return Run{}, nil, err
		}
	}

	closePosition := func(signalIndex int, price float64, executedAt time.Time, reason string) error {
		var fillPrice float64
		var quoteAmount float64
		var fee float64
		var tradePnL float64
		var quantity float64
		var side Side

		if baseBalance > 0 {
			fillPrice = applySellSlippage(price, request.SlippageRate)
			quantity = baseBalance
			quoteAmount = quantity * fillPrice
			fee = quoteAmount * request.FeeRate
			totalFees += fee
			estimatedSlippageCost += math.Abs(price-fillPrice) * quantity
			netQuoteAmount := quoteAmount - fee
			quoteBalance += netQuoteAmount
			tradePnL = netQuoteAmount - entryQuote
			side = SideSell
			sellCount++
		} else if baseBalance < 0 {
			fillPrice = applyBuySlippage(price, request.SlippageRate)
			quantity = math.Abs(baseBalance)
			costToBuy := quantity * fillPrice
			fee = costToBuy * request.FeeRate
			totalFees += fee
			estimatedSlippageCost += math.Abs(price-fillPrice) * quantity
			quoteBalance -= (costToBuy + fee)
			tradePnL = entryQuote - (costToBuy + fee)
			side = SideBuy
			buyCount++
			quoteAmount = costToBuy
		} else {
			return fmt.Errorf("cannot close position: no open position")
		}

		baseBalance = 0
		entryIndex = -1
		lastTradeIndex = signalIndex
		stopLossPrice = 0
		takeProfitPrice = 0
		completedTrades++

		if completedTrades == 1 || tradePnL > bestTrade {
			bestTrade = tradePnL
		}
		if completedTrades == 1 || tradePnL < worstTrade {
			worstTrade = tradePnL
		}
		if tradePnL > 0 {
			wins++
			winTotal += tradePnL
		} else if tradePnL < 0 {
			losses++
			lossTotal += tradePnL
		}
		if len(trades) == 0 {
			return fmt.Errorf("cannot close position without an entry trade")
		}
		trade := Trade{
			Symbol:      request.Symbol,
			Side:        side,
			Quantity:    quantity,
			Price:       fillPrice,
			QuoteAmount: quoteAmount,
			Fee:         fee,
			Equity:      quoteBalance,
			ExecutedAt:  executedAt,
		}
		trades = append(trades, trade)
		if openTrade != nil {
			var gross float64
			if openTrade.Side == SideBuy {
				gross = (fillPrice - openTrade.Price) * openTrade.Quantity
				if entryReferencePrice > 0 {
					grossProfitLoss += (price - entryReferencePrice) * openTrade.Quantity
				}
			} else {
				gross = (openTrade.Price - fillPrice) * openTrade.Quantity
				if entryReferencePrice > 0 {
					grossProfitLoss += (entryReferencePrice - price) * openTrade.Quantity
				}
			}
			fees := openTrade.Fee + fee
			roundTrips = append(roundTrips, RoundTrip{
				Symbol:          request.Symbol,
				EntryTime:       openTrade.ExecutedAt,
				ExitTime:        trade.ExecutedAt,
				EntryPrice:      openTrade.Price,
				ExitPrice:       fillPrice,
				Quantity:        openTrade.Quantity,
				GrossProfitLoss: gross,
				Fees:            fees,
				NetProfitLoss:   tradePnL,
				ProfitPercent:   tradePnL / entryQuote * 100,
				HoldingSeconds:  int64(trade.ExecutedAt.Sub(openTrade.ExecutedAt).Seconds()),
				EntryReason:     fmt.Sprintf("strategy %s signal", string(openTrade.Side)),
				ExitReason:      reason,
			})
			openTrade = nil
		}
		entryReferencePrice = 0
		return nil
	}

	for index := required - 1; index < len(candles); index++ {
		window := candles[:index+1]
		var signal strategy.Signal
		if signalSeries != nil {
			signal = signalSeries[index]
		} else {
			signal, err = evaluator.Evaluate(window)
			if err != nil {
				return Run{}, nil, err
			}
		}
		price, err := closePrice(candles[index])
		if err != nil {
			return Run{}, nil, err
		}

		tempLastTradeIndex := lastTradeIndex

		// Track whether an ATR exit fired on this candle to prevent
		// same-candle re-entry, which compounds fee losses.
		atrExitedThisCandle := false

		if baseBalance != 0 && index != entryIndex && canExitAfterMinimumHold(index, entryIndex, request.MinHoldingBars) {
			isShort := baseBalance < 0
			exitPrice, exitReason, ok, err := atrExit(request, candles[index], stopLossPrice, takeProfitPrice, isShort)
			if err != nil {
				return Run{}, nil, err
			}
			if ok {
				if err := closePosition(index, exitPrice, candles[index].CloseTime, exitReason); err != nil {
					return Run{}, nil, err
				}
				atrExitedThisCandle = true
			}
		}

		// Process exits from strategy signals
		if baseBalance > 0 && signal.Side == strategy.SideSell && canExitAfterMinimumHold(index, entryIndex, request.MinHoldingBars) {
			fillIndex, ok := strategyFillIndex(request, index, len(candles))
			if ok {
				fillPriceBase, executedAt, err := strategyFill(request, candles[fillIndex])
				if err == nil {
					if err := closePosition(fillIndex, fillPriceBase, executedAt, "strategy sell signal"); err != nil {
						return Run{}, nil, err
					}
				}
			}
		} else if baseBalance < 0 && signal.Side == strategy.SideBuy && canExitAfterMinimumHold(index, entryIndex, request.MinHoldingBars) {
			fillIndex, ok := strategyFillIndex(request, index, len(candles))
			if ok {
				fillPriceBase, executedAt, err := strategyFill(request, candles[fillIndex])
				if err == nil {
					if err := closePosition(fillIndex, fillPriceBase, executedAt, "strategy buy signal"); err != nil {
						return Run{}, nil, err
					}
				}
			}
		}

		// Process entries from strategy signals
		if baseBalance == 0 && !atrExitedThisCandle && canEnterAfterCooldown(index, tempLastTradeIndex, request.CooldownBars) {
			if signal.Side == strategy.SideBuy && quoteBalance > 0 && passesEntryFilters(request, candles[:index+1], price, false) {
				fillIndex, ok := strategyFillIndex(request, index, len(candles))
				if ok {
					fillPriceBase, executedAt, err := strategyFill(request, candles[fillIndex])
					if err == nil {
						quoteAmount := entryQuoteAmount(request, quoteBalance, baseBalance, price)
						if quoteAmount > 0 {
							fillPrice := applyBuySlippage(fillPriceBase, request.SlippageRate)
							fee := quoteAmount * request.FeeRate
							quantity := (quoteAmount - fee) / fillPrice
							if quantity > 0 {
								totalFees += fee
								estimatedSlippageCost += math.Abs(fillPriceBase-fillPrice) * quantity
								baseBalance = quantity
								quoteBalance -= quoteAmount
								entryQuote = quoteAmount
								entryReferencePrice = fillPriceBase
								entryIndex = fillIndex
								lastTradeIndex = fillIndex
								stopLossPrice, takeProfitPrice, err = atrExitPrices(request, candles[:index+1], fillPrice, false)
								if err == nil {
									buyCount++
									trade := Trade{
										Symbol:      request.Symbol,
										Side:        SideBuy,
										Quantity:    quantity,
										Price:       fillPrice,
										QuoteAmount: quoteAmount,
										Fee:         fee,
										Equity:      equity(quoteBalance, baseBalance, price),
										ExecutedAt:  executedAt,
									}
									trades = append(trades, trade)
									openTrade = &trades[len(trades)-1]
								}
							}
						}
					}
				}
			} else if signal.Side == strategy.SideSell && quoteBalance > 0 && request.ShortingEnabled && passesEntryFilters(request, candles[:index+1], price, true) {
				fillIndex, ok := strategyFillIndex(request, index, len(candles))
				if ok {
					fillPriceBase, executedAt, err := strategyFill(request, candles[fillIndex])
					if err == nil {
						quoteAmount := entryQuoteAmount(request, quoteBalance, baseBalance, price)
						if quoteAmount > 0 {
							fillPrice := applySellSlippage(fillPriceBase, request.SlippageRate)
							fee := quoteAmount * request.FeeRate
							quantity := (quoteAmount - fee) / fillPrice
							if quantity > 0 {
								totalFees += fee
								estimatedSlippageCost += math.Abs(fillPriceBase-fillPrice) * quantity
								baseBalance = -quantity
								quoteBalance += (quoteAmount - fee)
								entryQuote = quoteAmount - fee
								entryReferencePrice = fillPriceBase
								entryIndex = fillIndex
								lastTradeIndex = fillIndex
								stopLossPrice, takeProfitPrice, err = atrExitPrices(request, candles[:index+1], fillPrice, true)
								if err == nil {
									sellCount++
									trade := Trade{
										Symbol:      request.Symbol,
										Side:        SideSell,
										Quantity:    quantity,
										Price:       fillPrice,
										QuoteAmount: quoteAmount,
										Fee:         fee,
										Equity:      equity(quoteBalance, baseBalance, price),
										ExecutedAt:  executedAt,
									}
									trades = append(trades, trade)
									openTrade = &trades[len(trades)-1]
								}
							}
						}
					}
				}
			}
		}

		currentEquity := equity(quoteBalance, baseBalance, price)
		if currentEquity > peakEquity {
			peakEquity = currentEquity
		}
		if peakEquity > 0 {
			drawdown := (peakEquity - currentEquity) / peakEquity * 100
			maxDrawdown = math.Max(maxDrawdown, drawdown)
			equityCurve = append(equityCurve, EquityPoint{
				Time:            candles[index].CloseTime,
				Equity:          currentEquity,
				DrawdownPercent: drawdown,
			})
		}
	}

	if baseBalance != 0 {
		finalIndex := len(candles) - 1
		finalPrice, err := closePrice(candles[finalIndex])
		if err != nil {
			return Run{}, nil, err
		}
		if err := closePosition(finalIndex, finalPrice, candles[finalIndex].CloseTime, "end_of_backtest"); err != nil {
			return Run{}, nil, err
		}
		finalEquity := quoteBalance
		if finalEquity > peakEquity {
			peakEquity = finalEquity
		}
		if peakEquity > 0 {
			drawdown := (peakEquity - finalEquity) / peakEquity * 100
			maxDrawdown = math.Max(maxDrawdown, drawdown)
			equityCurve = append(equityCurve, EquityPoint{
				Time:            candles[finalIndex].CloseTime,
				Equity:          finalEquity,
				DrawdownPercent: drawdown,
			})
		}
	}

	lastPrice, err := closePrice(candles[len(candles)-1])
	if err != nil {
		return Run{}, nil, err
	}
	endingBalance := equity(quoteBalance, baseBalance, lastPrice)
	profitLoss := endingBalance - request.StartingBalance
	winRate := 0.0
	if completedTrades > 0 {
		winRate = float64(wins) / float64(completedTrades) * 100
	}
	averageWin := 0.0
	if wins > 0 {
		averageWin = winTotal / float64(wins)
	}
	averageLoss := 0.0
	if losses > 0 {
		averageLoss = lossTotal / float64(losses)
	}
	profitFactor := 0.0
	if lossTotal < 0 {
		profitFactor = winTotal / math.Abs(lossTotal)
	} else if winTotal > 0 {
		profitFactor = 999999
	}
	averageTrade := 0.0
	averageHoldingSeconds := 0.0
	if completedTrades > 0 {
		averageTrade = (winTotal + lossTotal) / float64(completedTrades)
		totalHoldingSeconds := int64(0)
		for _, roundTrip := range roundTrips {
			totalHoldingSeconds += roundTrip.HoldingSeconds
		}
		if len(roundTrips) > 0 {
			averageHoldingSeconds = float64(totalHoldingSeconds) / float64(len(roundTrips))
		}
	}
	expectancy := averageTrade
	tradesPerDay := tradesPerDay(request.From, request.To, len(trades))
	evaluatedCandles := len(candles) - required + 1
	churnRatio := churnRatio(len(trades), evaluatedCandles)
	sharpeRatio, sortinoRatio := riskAdjustedRatios(equityCurve, request.Interval)
	benchmarkEndingBalance, benchmarkProfitLoss, benchmarkReturnPercent, err := benchmarkBuyAndHold(request, candles)
	if err != nil {
		return Run{}, nil, err
	}
	returnPercent := profitLoss / request.StartingBalance * 100
	roundTripCostPercent := (request.FeeRate*2 + request.SlippageRate*2) * 100
	breakEvenMovePercent := roundTripCostPercent
	excessReturnPercent := returnPercent - benchmarkReturnPercent
	validationStatus, validationReason := validateRunCandidate(runValidationInput{
		Interval:               request.Interval,
		From:                   request.From,
		To:                     request.To,
		CompletedTrades:        completedTrades,
		TotalTrades:            len(trades),
		ProfitFactor:           profitFactor,
		MaxDrawdown:            maxDrawdown,
		AverageTrade:           averageTrade,
		BreakEvenMovePercent:   breakEvenMovePercent,
		ReturnPercent:          returnPercent,
		BenchmarkReturnPercent: benchmarkReturnPercent,
		ExcessReturnPercent:    excessReturnPercent,
		ExecutionFillMode:      request.ExecutionFillMode,
		TradesPerDay:           tradesPerDay,
	})

	run := Run{
		StrategyName:            request.StrategyName,
		Version:                 request.Version,
		Symbol:                  request.Symbol,
		Interval:                request.Interval,
		From:                    request.From,
		To:                      request.To,
		FastPeriod:              request.FastPeriod,
		SlowPeriod:              request.SlowPeriod,
		RSIPeriod:               request.RSIPeriod,
		RSIOversold:             request.RSIOversold,
		RSIOverbought:           request.RSIOverbought,
		StartingBalance:         request.StartingBalance,
		EndingBalance:           endingBalance,
		ProfitLoss:              profitLoss,
		GrossProfitLoss:         grossProfitLoss,
		TotalFees:               totalFees,
		EstimatedSlippageCost:   estimatedSlippageCost,
		RoundTripCostPercent:    roundTripCostPercent,
		BreakEvenMovePercent:    breakEvenMovePercent,
		ReturnPercent:           returnPercent,
		WinRate:                 winRate,
		MaxDrawdown:             maxDrawdown,
		TotalTrades:             len(trades),
		BuyCount:                buyCount,
		SellCount:               sellCount,
		BestTrade:               bestTrade,
		WorstTrade:              worstTrade,
		AverageWin:              averageWin,
		AverageLoss:             averageLoss,
		OpenPosition:            baseBalance > 0,
		FeeRate:                 request.FeeRate,
		SlippageRate:            request.SlippageRate,
		PositionSizingMode:      request.PositionSizingMode,
		PositionSizeValue:       request.PositionSizeValue,
		TrendFilterEnabled:      request.TrendFilterEnabled,
		TrendPeriod:             request.TrendPeriod,
		CooldownBars:            request.CooldownBars,
		MinHoldingBars:          request.MinHoldingBars,
		ATRExitEnabled:          request.ATRExitEnabled,
		ATRPeriod:               request.ATRPeriod,
		ATRStopMultiplier:       request.ATRStopMultiplier,
		ATRTakeProfitMultiplier: request.ATRTakeProfitMultiplier,
		RegimeFilterEnabled:     request.RegimeFilterEnabled,
		RegimeFilterPeriod:      request.RegimeFilterPeriod,
		RegimeMinATRPercent:     request.RegimeMinATRPercent,
		RegimeMaxATRPercent:     request.RegimeMaxATRPercent,
		ShortingEnabled:         request.ShortingEnabled,
		WinningTrades:           wins,
		LosingTrades:            losses,
		ProfitFactor:            profitFactor,
		AverageTrade:            averageTrade,
		AverageHoldingSeconds:   averageHoldingSeconds,
		Expectancy:              expectancy,
		TradesPerDay:            tradesPerDay,
		ChurnRatio:              churnRatio,
		SharpeRatio:             sharpeRatio,
		SortinoRatio:            sortinoRatio,
		BenchmarkEndingBalance:  benchmarkEndingBalance,
		BenchmarkProfitLoss:     benchmarkProfitLoss,
		BenchmarkReturnPercent:  benchmarkReturnPercent,
		ExcessReturnPercent:     excessReturnPercent,
		ValidationStatus:        validationStatus,
		ValidationReason:        validationReason,
		ExecutionFillMode:       request.ExecutionFillMode,
		RoundTrips:              roundTrips,
		EquityCurve:             equityCurve,
	}
	return run, trades, nil
}

func atrExitPrices(request Request, candles []marketdata.Candle, entryPrice float64, isShort bool) (float64, float64, error) {
	if !request.ATRExitEnabled {
		return 0, 0, nil
	}
	value, err := indicator.AverageTrueRange(candles, request.ATRPeriod)
	if err != nil {
		return 0, 0, err
	}
	stopLoss := 0.0
	if request.ATRStopMultiplier > 0 {
		if isShort {
			stopLoss = entryPrice + value*request.ATRStopMultiplier
		} else {
			stopLoss = entryPrice - value*request.ATRStopMultiplier
		}
	}
	takeProfit := 0.0
	if request.ATRTakeProfitMultiplier > 0 {
		if isShort {
			takeProfit = entryPrice - value*request.ATRTakeProfitMultiplier
		} else {
			takeProfit = entryPrice + value*request.ATRTakeProfitMultiplier
		}
	}
	return stopLoss, takeProfit, nil
}

func atrExit(request Request, candle marketdata.Candle, stopLoss float64, takeProfit float64, isShort bool) (float64, string, bool, error) {
	if !request.ATRExitEnabled {
		return 0, "", false, nil
	}
	low, err := candleLow(candle)
	if err != nil {
		return 0, "", false, err
	}
	high, err := candleHigh(candle)
	if err != nil {
		return 0, "", false, err
	}
	if isShort {
		if stopLoss > 0 && high >= stopLoss {
			return stopLoss, "ATR stop-loss", true, nil
		}
		if takeProfit > 0 && low <= takeProfit {
			return takeProfit, "ATR take-profit", true, nil
		}
	} else {
		if stopLoss > 0 && low <= stopLoss {
			return stopLoss, "ATR stop-loss", true, nil
		}
		if takeProfit > 0 && high >= takeProfit {
			return takeProfit, "ATR take-profit", true, nil
		}
	}
	return 0, "", false, nil
}

func closePrice(candle marketdata.Candle) (float64, error) {
	price, err := strconv.ParseFloat(candle.Close, 64)
	if err != nil {
		return 0, fmt.Errorf("parse candle close %q: %w", candle.Close, err)
	}
	if price <= 0 {
		return 0, fmt.Errorf("candle close must be positive")
	}
	return price, nil
}

func candleHigh(candle marketdata.Candle) (float64, error) {
	if candle.High == "" {
		return closePrice(candle)
	}
	price, err := strconv.ParseFloat(candle.High, 64)
	if err != nil {
		return 0, fmt.Errorf("parse candle high %q: %w", candle.High, err)
	}
	if price <= 0 {
		return 0, fmt.Errorf("candle high must be positive")
	}
	return price, nil
}

func candleLow(candle marketdata.Candle) (float64, error) {
	if candle.Low == "" {
		return closePrice(candle)
	}
	price, err := strconv.ParseFloat(candle.Low, 64)
	if err != nil {
		return 0, fmt.Errorf("parse candle low %q: %w", candle.Low, err)
	}
	if price <= 0 {
		return 0, fmt.Errorf("candle low must be positive")
	}
	return price, nil
}

func openPrice(candle marketdata.Candle) (float64, error) {
	if candle.Open == "" {
		return 0, fmt.Errorf("candle open is required for next_open fill mode")
	}
	price, err := strconv.ParseFloat(candle.Open, 64)
	if err != nil {
		return 0, fmt.Errorf("parse candle open %q: %w", candle.Open, err)
	}
	if price <= 0 {
		return 0, fmt.Errorf("candle open must be positive")
	}
	return price, nil
}

func strategyFillIndex(request Request, signalIndex int, candleCount int) (int, bool) {
	if request.ExecutionFillMode == ExecutionFillModeNextOpen {
		nextIndex := signalIndex + 1
		return nextIndex, nextIndex < candleCount
	}
	return signalIndex, true
}

func strategyFill(request Request, candle marketdata.Candle) (float64, time.Time, error) {
	if request.ExecutionFillMode == ExecutionFillModeNextOpen {
		price, err := openPrice(candle)
		return price, candle.OpenTime, err
	}
	price, err := closePrice(candle)
	return price, candle.CloseTime, err
}

func equity(quoteBalance float64, baseBalance float64, price float64) float64 {
	return quoteBalance + baseBalance*price
}

func entryQuoteAmount(request Request, quoteBalance float64, baseBalance float64, price float64) float64 {
	switch request.PositionSizingMode {
	case PositionSizingFixedQuote:
		return math.Min(request.PositionSizeValue, quoteBalance)
	case PositionSizingPercentEquity:
		target := equity(quoteBalance, baseBalance, price) * request.PositionSizeValue / 100
		return math.Min(target, quoteBalance)
	default:
		return quoteBalance
	}
}

func canEnterAfterCooldown(currentIndex int, lastTradeIndex int, cooldownBars int) bool {
	return cooldownBars <= 0 || lastTradeIndex < 0 || currentIndex-lastTradeIndex >= cooldownBars
}

func canExitAfterMinimumHold(currentIndex int, entryIndex int, minHoldingBars int) bool {
	return minHoldingBars <= 0 || entryIndex < 0 || currentIndex-entryIndex >= minHoldingBars
}

func passesEntryFilters(request Request, candles []marketdata.Candle, price float64, isShort bool) bool {
	return passesTrendFilter(request, candles, price, isShort) && passesRegimeFilter(request, candles, price)
}

func passesTrendFilter(request Request, candles []marketdata.Candle, price float64, isShort bool) bool {
	if !request.TrendFilterEnabled || request.StrategyName != strategy.StrategySMACrossover {
		return true
	}
	if request.TrendPeriod <= 0 || len(candles) < request.TrendPeriod {
		return false
	}
	trendAverage, err := averageCandleClose(candles[len(candles)-request.TrendPeriod:])
	if err != nil {
		return false
	}
	if isShort {
		return price < trendAverage
	}
	return price > trendAverage
}

func passesRegimeFilter(request Request, candles []marketdata.Candle, price float64) bool {
	if !request.RegimeFilterEnabled {
		return true
	}
	if request.RegimeFilterPeriod <= 0 || len(candles) < request.RegimeFilterPeriod+1 || price <= 0 {
		return false
	}
	atrValue, err := indicator.AverageTrueRange(candles, request.RegimeFilterPeriod)
	if err != nil {
		return false
	}
	atrPercent := atrValue / price * 100
	if request.RegimeMinATRPercent > 0 && atrPercent < request.RegimeMinATRPercent {
		return false
	}
	if request.RegimeMaxATRPercent > 0 && atrPercent > request.RegimeMaxATRPercent {
		return false
	}
	return true
}

func averageCandleClose(candles []marketdata.Candle) (float64, error) {
	if len(candles) == 0 {
		return 0, fmt.Errorf("no candles to average")
	}
	total := 0.0
	for _, candle := range candles {
		price, err := closePrice(candle)
		if err != nil {
			return 0, err
		}
		total += price
	}
	return total / float64(len(candles)), nil
}

func benchmarkBuyAndHold(request Request, candles []marketdata.Candle) (float64, float64, float64, error) {
	firstPrice, err := closePrice(candles[0])
	if err != nil {
		return 0, 0, 0, err
	}
	lastPrice, err := closePrice(candles[len(candles)-1])
	if err != nil {
		return 0, 0, 0, err
	}
	fillPrice := applyBuySlippage(firstPrice, request.SlippageRate)
	fee := request.StartingBalance * request.FeeRate
	quantity := (request.StartingBalance - fee) / fillPrice
	exitPrice := applySellSlippage(lastPrice, request.SlippageRate)
	exitGross := quantity * exitPrice
	exitFee := exitGross * request.FeeRate
	endingBalance := exitGross - exitFee
	profitLoss := endingBalance - request.StartingBalance
	returnPercent := profitLoss / request.StartingBalance * 100
	return endingBalance, profitLoss, returnPercent, nil
}

func applyBuySlippage(price float64, slippageRate float64) float64 {
	return price * (1 + slippageRate)
}

func applySellSlippage(price float64, slippageRate float64) float64 {
	return price * (1 - slippageRate)
}

func tradesPerDay(from time.Time, to time.Time, totalTrades int) float64 {
	if totalTrades <= 0 || from.IsZero() || to.IsZero() || !to.After(from) {
		return 0
	}
	days := to.Sub(from).Hours() / 24
	if days <= 0 {
		return 0
	}
	return float64(totalTrades) / days
}

func churnRatio(totalTrades int, evaluatedCandles int) float64 {
	if totalTrades <= 0 || evaluatedCandles <= 0 {
		return 0
	}
	return float64(totalTrades) / float64(evaluatedCandles) * 100
}

func riskAdjustedRatios(points []EquityPoint, interval string) (float64, float64) {
	if len(points) < 2 {
		return 0, 0
	}
	returns := make([]float64, 0, len(points)-1)
	downsideSquares := []float64{}
	for index := 1; index < len(points); index++ {
		previous := points[index-1].Equity
		if previous <= 0 {
			continue
		}
		value := (points[index].Equity - previous) / previous
		returns = append(returns, value)
		if value < 0 {
			downsideSquares = append(downsideSquares, value*value)
		}
	}
	if len(returns) < 2 {
		return 0, 0
	}
	meanReturn := mean(returns)
	standardDeviation := stddev(returns, meanReturn)
	annualization := annualizationFactor(interval)
	sharpe := 0.0
	if standardDeviation > 0 {
		sharpe = meanReturn / standardDeviation * math.Sqrt(annualization)
	}
	sortino := 0.0
	if len(downsideSquares) > 0 {
		downsideDeviation := math.Sqrt(mean(downsideSquares))
		if downsideDeviation > 0 {
			sortino = meanReturn / downsideDeviation * math.Sqrt(annualization)
		}
	}
	return sharpe, sortino
}

func annualizationFactor(interval string) float64 {
	duration, err := marketdata.IntervalDuration(interval)
	if err != nil || duration <= 0 {
		return 1
	}
	return (365 * 24 * time.Hour).Seconds() / duration.Seconds()
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

func stddev(values []float64, average float64) float64 {
	if len(values) < 2 {
		return 0
	}
	total := 0.0
	for _, value := range values {
		diff := value - average
		total += diff * diff
	}
	return math.Sqrt(total / float64(len(values)-1))
}

type runValidationInput struct {
	Interval               string
	From                   time.Time
	To                     time.Time
	CompletedTrades        int
	TotalTrades            int
	ProfitFactor           float64
	MaxDrawdown            float64
	AverageTrade           float64
	BreakEvenMovePercent   float64
	ReturnPercent          float64
	BenchmarkReturnPercent float64
	ExcessReturnPercent    float64
	ExecutionFillMode      string
	TradesPerDay           float64
}

func validateRunCandidate(input runValidationInput) (string, string) {
	if input.BenchmarkReturnPercent >= 20 && input.ReturnPercent < input.BenchmarkReturnPercent*0.25 {
		return "low_bull_market_capture", "strategy captures too little of a strong positive benchmark move"
	}
	if input.CompletedTrades < 100 {
		return "insufficient_sample", "completed trades below 100"
	}
	if input.TradesPerDay > 50 && input.Interval == "1m" {
		return "overtrading", "trade frequency is too high for the selected interval"
	}
	if input.BreakEvenMovePercent > 0 && input.AverageTrade > 0 && input.AverageTrade < input.BreakEvenMovePercent {
		return "cost_drag", "average trade does not exceed estimated round-trip cost"
	}
	if input.ExcessReturnPercent <= 0 {
		return "underperforms_benchmark", "excess return must be positive"
	}
	if input.ProfitFactor <= 1.2 {
		return "weak_profit_factor", "profit factor must be greater than 1.2"
	}
	if input.MaxDrawdown > 30 {
		return "high_drawdown", "maximum drawdown above 30%"
	}
	if input.AverageTrade <= 0 {
		return "negative_average_trade", "average completed trade must be positive after fees"
	}
	if exceedsTradeFrequencyLimit(input.Interval, input.From, input.To, input.TotalTrades) {
		return "overtrading", "trade frequency is too high for the selected interval"
	}
	if input.ExecutionFillMode != ExecutionFillModeNextOpen {
		return "unsafe_execution_timing", "candidate backtests must use next_open execution"
	}
	return "candidate", "strategy meets hardened validation rules"
}

func exceedsTradeFrequencyLimit(interval string, from time.Time, to time.Time, totalTrades int) bool {
	if totalTrades <= 0 || from.IsZero() || to.IsZero() || !to.After(from) {
		return false
	}
	days := to.Sub(from).Hours() / 24
	if days <= 0 {
		return false
	}
	return float64(totalTrades)/days > maxTradesPerDay(interval)
}

func maxTradesPerDay(interval string) float64 {
	switch interval {
	case "1m":
		return 20
	case "5m":
		return 12
	case "15m":
		return 4
	case "1h":
		return 2
	default:
		return 6
	}
}
