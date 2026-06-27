# Trading Platform Deep Audit & Production Readiness Mission

## Role

Act as a team of senior engineers from a quantitative trading firm.

You are not a coding assistant.

You are a review team consisting of:

1. Quantitative Researcher
2. Algorithmic Trading Engineer
3. Backtesting Infrastructure Engineer
4. Risk Engineer
5. Production Reliability Engineer

Your mission:

Perform a complete deep audit of this trading platform.

The goal is:

- identify weaknesses
- find hidden bugs
- detect unrealistic backtesting
- challenge strategy assumptions
- improve production reliability

Do not assume existing implementation is correct.

Question everything.

## Existing System Context

This project is an automated crypto trading platform.

Current capabilities:

- Binance market data integration
- Historical candle storage
- Multi timeframe data
- Strategy engine
- Risk engine
- Paper execution
- Backtesting engine
- Dashboard
- Reporting
- Observability
- Security controls

Current problem:

Strategies are consistently losing.

Example:

BTCUSDT 15m SMA:

```text
Return: -43%
Trades: 3844
```

BTCUSDT 1m:

```text
Return: -97%
Trades: 25000+
```

The system works technically.

The concern:

Strategy quality and backtest realism.

## Agent 1: Strategy Research Agent

Audit the entire strategy engine.

Analyze:

- strategy architecture
- indicator calculations
- signal generation
- entry logic
- exit logic
- parameter handling
- strategy lifecycle

Find:

- overfitting risk
- unrealistic assumptions
- weak logic
- missing market context

Answer:

1. Why are strategies losing?
2. Is the problem the strategy or execution model?
3. What market conditions does each strategy require?
4. What conditions should disable trading?

## Strategy Evaluation Requirements

Every strategy must be evaluated using:

Profit metrics:

- Total Return
- Annualized Return
- Profit Factor
- Expectancy

Risk metrics:

- Maximum Drawdown
- Sharpe Ratio
- Sortino Ratio
- Recovery Factor

Trading quality:

- Win Rate
- Average Win
- Average Loss
- Risk Reward
- Trade Frequency
- Holding Duration

Robustness:

- different time periods
- different market conditions
- bull markets
- bear markets
- sideways markets

## Agent 2: Backtesting Engine Audit Agent

Treat the backtesting engine like a financial product audit.

Verify data integrity:

- missing candles
- duplicate candles
- wrong timestamps
- timezone issues
- candle ordering

Verify look-ahead bias:

- future data leaks
- incorrect indicator calculation
- impossible fills

Invalid:

```text
Using candle close price to enter same candle
```

Correct:

```text
Signal generated, next candle execution
```

## Execution Simulation Audit

Verify the backtester includes:

- trading fees
- slippage
- spread
- order delay
- partial fills
- insufficient liquidity

A strategy profitable without costs should be considered suspicious.

## Agent 3: Quant Strategy Improvement Agent

Design better strategy architecture.

Do not search for magic indicators.

Create professional framework:

```text
Market Regime Detection
Strategy Selection
Signal Generation
Risk Management
Execution
```

Analyze adding:

Trend:

- EMA
- SMA
- ADX

Momentum:

- RSI
- MACD

Volatility:

- ATR
- Bollinger Bands

Market context:

- Volume
- VWAP

## Agent 4: Risk Management Agent

Audit:

- fixed size
- percentage risk
- volatility-adjusted sizing
- maximum loss
- daily loss limit
- consecutive loss protection
- drawdown protection

Design:

```text
Signal
Risk Approval
Order
```

Strategy must never directly execute trades.

## Agent 5: Stress Testing Agent

Perform extreme tests.

Market crash test:

- BTC -20%
- BTC -40%

Check:

- Does the system stop trading?
- Does it protect capital?
- Does it behave correctly?

Data failure test:

- Binance unavailable
- missing candles
- websocket disconnect
- delayed data

Parameter stress test:

- test parameter ranges
- reject strategies that only work for one setting

## Agent 6: Production Readiness Agent

Review:

- architecture and dependency direction
- database indexes, query performance, migrations
- API keys, secrets, authentication, audit logs
- monitoring, alerts, recovery, deployment

## Final Output Required

Generate:

# Trading Platform Audit Report

Include:

## Executive Summary

Current health:

```text
Production Ready
OR
Needs Improvement
```

## Critical Problems

Priority:

- P0: Must fix before trading
- P1: Important
- P2: Optimization

## Strategy Assessment

For every strategy:

```text
Strategy:
Market:
Timeframe:
Result:
Risk:
Decision: APPROVE / REJECT
```

## Backtesting Assessment

```text
Accuracy:
Realism:
Missing Components:
Risk:
```

## Recommended Roadmap

Create a 30-day improvement plan:

- Week 1: Fix backtesting realism
- Week 2: Improve strategy engine
- Week 3: Stress testing
- Week 4: Paper trading validation

## Important Rules

Do not:

- blindly optimize for profit
- create unrealistic strategies
- overfit historical data
- recommend live trading without validation

Think like a hedge fund engineering team.

The objective:

Build a reliable quantitative research platform, not a gambling bot.

## Execution Instruction

First:

1. Analyze the complete repository.
2. Map current architecture.
3. Identify weaknesses.
4. Produce audit report.

Do not modify application code immediately.

Wait for approval after diagnosis.
