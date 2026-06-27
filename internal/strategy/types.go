package strategy

import (
	"time"
)

type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
	SideHold Side = "hold"
)

const (
	StrategySMACrossover     = "sma-crossover"
	StrategyRSIMeanReversion = "rsi-mean-reversion"
	StrategyBTCTrendPullback = "btc-trend-pullback"
	StrategyPineCustom       = "pine-custom"

	StrategyTrendFollowingMTF      = "trend-following-mtf"
	StrategyMomentumBreakoutVolume = "momentum-breakout-volume"
	StrategyAdaptiveMeanReversion  = "adaptive-mean-reversion"
	StrategyStatArbPairs           = "stat-arb-pairs"
	StrategyVWAPReversion          = "vwap-reversion"
	StrategyCryptoMarketMaking     = "crypto-market-making"
	StrategyFundingRateArbitrage   = "funding-rate-arbitrage"
	StrategyGridTrading            = "grid-trading"
	StrategyMultiFactorMomentum    = "multi-factor-momentum"
	StrategySmartMoneyOrderFlow    = "smart-money-order-flow"
)

type Signal struct {
	ID           string
	StrategyName string
	Version      string
	Symbol       string
	Interval     string
	Side         Side
	Strength     float64
	Reason       string
	GeneratedAt  time.Time
}
