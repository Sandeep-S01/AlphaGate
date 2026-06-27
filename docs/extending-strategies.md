# Extending Sentra with New Strategies and Indicators

Sentra is designed to be extensible, allowing developers to add new trading strategies and technical indicators without modifying core system components. This document explains how to extend Sentra with new strategies.

## Overview

The strategy system in Sentra follows a plugin-like architecture where new strategies can be added by implementing the `strategy.Evaluator` interface. The system uses dependency injection and configuration-driven behavior to make extension straightforward.

## Core Concepts

### Predefined Strategy Templates

Sentra ships a static predefined strategy catalog from `internal/strategy/templates.go`.
Templates are not the same thing as live runtime evaluators:

| Status | Meaning |
| --- | --- |
| `executable_native` | The template maps directly to a Go evaluator. |
| `executable_pine` | The template includes Pine code that the current Pine parser can validate and save. |
| `template_only` | The template is available for review/copying, but some rules need native implementation or richer parser support before live execution. |
| `blocked_by_data` | The strategy needs data or execution infrastructure Sentra does not currently collect, such as L2 books, funding rates, open interest, liquidation feeds, or multi-leg execution. |

The template API is read-only:

```text
GET /api/v1/strategies/templates
GET /api/v1/strategies/templates/{id}
```

Current predefined templates:

| ID | Strategy | Status | Required Data |
| --- | --- | --- | --- |
| `trend-following-mtf` | Trend Following (Multi-Timeframe) | `executable_pine` | OHLCV, 20-period highs/lows |
| `momentum-breakout-volume` | Momentum Breakout (Volume-Confirmed) | `executable_pine` | OHLCV, resistance context |
| `adaptive-mean-reversion` | Adaptive Mean Reversion | `executable_pine` | OHLCV |
| `stat-arb-pairs` | Statistical Arbitrage / Pairs Trading | `blocked_by_data` | synchronized two-symbol OHLCV, cointegration inputs |
| `vwap-reversion` | VWAP Reversion | `template_only` | intraday OHLCV, session calendar |
| `crypto-market-making` | Market Making (Crypto-Focused) | `blocked_by_data` | L1/L2 book, inventory, funding, latency telemetry |
| `funding-rate-arbitrage` | Funding Rate Arbitrage | `blocked_by_data` | spot/perp prices, funding rates, borrow rates |
| `grid-trading` | Grid Trading | `template_only` | OHLCV, persistent grid order state |
| `multi-factor-momentum` | Multi-Factor Momentum | `executable_pine` | OHLCV |
| `smart-money-order-flow` | Smart Money / Order Flow | `blocked_by_data` | tick data, CVD, L2 depth, liquidations, OI, funding |

To promote a template into a native evaluator:

1. Confirm the required data exists in Sentra repositories.
2. Add or reuse indicator helpers with focused unit tests.
3. Implement `strategy.Evaluator` in `internal/strategy/`.
4. Add the strategy constant, settings validation, required-candle calculation, and factory mapping.
5. Add backtest validation if the strategy has special execution semantics.
6. Change the template status from `template_only` or `blocked_by_data` to `executable_native` only after the evaluator is covered by tests.

### The Evaluator Interface

All strategies must implement the `strategy.Evaluator` interface:

```go
type Evaluator interface {
    Evaluate(candles []marketdata.Candle) (Signal, error)
}
```

The `Evaluate` method receives historical candle data and returns a trading signal.

### Signal Structure

Trading signals have the following structure:

```go
type Signal struct {
    ID           string
    StrategyName string
    Version      string
    Symbol       string
    Interval     string
    Side         Side // "buy", "sell", or "hold"
    Strength     float64 // Normalized strength (0-100 or higher)
    Reason       string // Human-readable explanation
    GeneratedAt  time.Time
}
```

### Configuration Flow

1. Strategy configuration is stored in the `strategy_settings` database table
2. Configuration is loaded into a `strategy.Settings` struct
3. The factory function `NewEvaluatorFromSettings` creates the appropriate strategy instance
4. Settings include validation logic to ensure parameters are correct

## Adding a New Strategy

Here's a step-by-step guide to adding a new strategy to Sentra.

### Step 1: Choose a Strategy Name

Pick a unique name for your strategy and add it to the constants in `internal/strategy/types.go`:

```go
const (
    StrategySMACrossover     = "sma-crossover"
    StrategyRSIMeanReversion = "rsi-mean-reversion"
    StrategyBTCTrendPullback = "btc-trend-pullback"
    StrategyPineCustom       = "pine-custom"
    StrategyYourNewStrategy  = "your-strategy-name" // Add your strategy here
)
```

### Step 2: Implement the Evaluator Interface

Create a new Go file in `internal/strategy/` for your strategy (e.g., `yourstrategy.go`):

```go
package strategy

import (
    "fmt"
    "time"

    "sentra/internal/marketdata"
)

// YourStrategyConfig holds the configuration for your strategy
type YourStrategyConfig struct {
    Name       string
    Version    string
    Symbol     string
    Interval   string
    // Add your strategy-specific parameters here
    Param1     int
    Param2     float64
}

// YourStrategy implements the Evaluator interface
type YourStrategy struct {
    cfg YourStrategyConfig
}

// NewYourStrategy creates a new instance of YourStrategy
func NewYourStrategy(cfg YourStrategyConfig) *YourStrategy {
    return &YourStrategy{cfg: cfg}
}

// Evaluate implements the Evaluator interface
func (s *YourStrategy) Evaluate(candles []marketdata.Candle) (Signal, error) {
    // Validate parameters
    if s.cfg.Param1 <= 0 {
        return Signal{}, fmt.Errorf("param1 must be positive")
    }

    // Check if we have enough candles
    required := s.calculateRequiredCandles()
    if len(candles) < required {
        return Signal{}, fmt.Errorf("insufficient candles: need %d, got %d", required, len(candles))
    }

    // Your strategy logic goes here
    // Analyze the candles and determine:
    // - side: SideBuy, SideSell, or SideHold
    // - strength: a float64 representing signal strength
    // - reason: a human-readable explanation

    side := SideHold
    strength := 0.0
    reason := "no signal"

    // Example logic (replace with your actual strategy):
    // if some condition {
    //     side = SideBuy
    //     strength = calculateStrength(...)
    //     reason = "explanation of why we're buying"
    // } else if some other condition {
    //     side = SideSell
    //     strength = calculateStrength(...)
    //     reason = "explanation of why we're selling"
    // }

    generatedAt := time.Now().UTC()
    if len(candles) > 0 {
        generatedAt = candles[len(candles)-1].CloseTime
    }

    return Signal{
        StrategyName: s.cfg.Name,
        Version:      s.cfg.Version,
        Symbol:       s.cfg.Symbol,
        Interval:     s.cfg.Interval,
        Side:         side,
        Strength:     strength,
        Reason:       reason,
        GeneratedAt:  generatedAt,
    }, nil
}

// calculateRequiredCandles returns the minimum number of candles needed
func (s *YourStrategy) calculateRequiredCandles() int {
    // Return the minimum number of candles your strategy needs
    // For example, if you need 20 periods for an indicator plus 1 for current:
    // return s.cfg.Param1 + 1
    return s.cfg.Param1 + 1 // Replace with your actual calculation
}
```

### Step 3: Update the Factory Function

Modify the `NewEvaluatorFromSettings` function in `internal/strategy/factory.go` to handle your new strategy:

```go
func NewEvaluatorFromSettings(settings Settings) (Evaluator, error) {
    settings = settings.Normalized()
    if err := settings.Validate(); err != nil {
        return nil, err
    }
    switch settings.StrategyName {
    case StrategySMACrossover:
        return NewSMACrossover(SMAConfig{
            // ... existing fields
        }), nil
    case StrategyRSIMeanReversion:
        return NewRSIMeanReversion(RSIConfig{
            // ... existing fields
        }), nil
    case StrategyBTCTrendPullback:
        return NewBTCTrendPullback(TrendPullbackConfig{
            // ... existing fields
        }), nil
    case StrategyPineCustom:
        // ... existing Pine custom logic
    case StrategyYourNewStrategy: // Add your strategy here
        return NewYourStrategy(YourStrategyConfig{
            Name:       settings.StrategyName,
            Version:    settings.Version,
            Symbol:     settings.Symbol,
            Interval:   settings.Interval,
            // Map your settings fields to your config struct
            Param1:     settings.FastPeriod,    // Example mapping
            Param2:     settings.SlowPeriod,    // Example mapping
        }), nil
    default:
        return nil, fmt.Errorf("unsupported strategy_name %q", settings.StrategyName)
    }
}
```

### Step 4: Add Validation Logic

Update the `Settings.Validate()` method in `internal/strategy/settings.go` to validate your strategy's parameters:

```go
func (s Settings) Validate() error {
    // ... existing validation code
    
    switch s.StrategyName {
    // ... existing case statements
    case StrategyYourNewStrategy:
        if s.FastPeriod <= 0 { // Example validation
            return fmt.Errorf("fast_period must be positive")
        }
        if s.SlowPeriod <= s.FastPeriod { // Example validation
            return fmt.Errorf("slow_period must be greater than fast_period")
        }
        if s.LookbackLimit < s.SlowPeriod { // Example validation
            return fmt.Errorf("lookback_limit must be greater than or equal to slow_period")
        }
        // Add more validation as needed for your strategy
    default:
        return fmt.Errorf("unsupported strategy_name %q", s.StrategyName)
    }
    
    return nil
}
```

### Step 5: Update Required Candles Calculation (if needed)

If your strategy has special requirements for the number of candles needed, update the `RequiredCandles()` method in `internal/strategy/settings.go`:

```go
func (s Settings) RequiredCandles() int {
    switch s.StrategyName {
    // ... existing case statements
    case StrategyYourNewStrategy:
        // Return the number of candles your strategy needs
        // This should match what your strategy's calculateRequiredCandles method returns
        return s.FastPeriod + 1 // Example - replace with your actual calculation
    default:
        return s.SlowPeriod + 1
    }
}
```

### Step 6: Update Database Query (if using default settings retrieval)

If you want the system to automatically load your strategy's settings from the database, update the SQL query in `internal/strategy/settings.go`:

```go
func (r *SettingsRepository) Get(ctx context.Context) (Settings, error) {
    query := `
    SELECT strategy_name, version, symbol, interval, fast_period, slow_period, lookback_limit,
           rsi_period, rsi_oversold, rsi_overbought, updated_at
    FROM strategy_settings
    WHERE strategy_name IN ('sma-crossover', 'rsi-mean-reversion', 'btc-trend-pullback', 'your-strategy-name') // Add your strategy here
    ORDER BY updated_at DESC
    LIMIT 1`
    
    // ... rest of the function remains the same
}
```

### Step 7: Add Migration for Default Settings (Optional)

If you want to provide default settings for your strategy, create a migration:

1. Create a new migration file: `migrations/NNNNNN_your_strategy_settings.up.sql`
2. Add the SQL to insert default settings:
   ```sql
   INSERT INTO strategy_settings (
       strategy_name, version, symbol, interval, fast_period, slow_period, lookback_limit,
       rsi_period, rsi_oversold, rsi_overbought, updated_at
   ) VALUES (
       'your-strategy-name', 'v1', 'BTCUSDT', '1m', 9, 21, 100,
       14, 30, 70, NOW()
   )
   ON CONFLICT (strategy_name) DO NOTHING;
   ```
3. Create the corresponding down migration to remove the settings if needed.

## Adding Custom Indicators

If your strategy requires custom technical indicators that aren't already available, you have a few options:

### Option 1: Implement Indicator Logic Directly in Your Strategy
For simple indicators, implement the calculation directly in your strategy's `Evaluate` method.

### Option 2: Create a Reusable Indicator Package
For more complex indicators that might be used by multiple strategies:
1. Create a new package (e.g., `internal/indicators`)
2. Implement your indicator functions there
3. Import and use them in your strategy

### Option 3: Extend the Pine Script Parser
Sentra already supports Pine Script custom strategies through the `pine` package. If your strategy can be expressed in Pine Script, consider adding it as a Pine Script strategy instead of a native Go strategy.

## Testing Your Strategy

### Unit Tests
Create a test file for your strategy (e.g., `yourstrategy_test.go`):

```go
package strategy

import (
    "testing"
    "time"

    "sentra/internal/marketdata"
)

func TestYourStrategy_Evaluate(t *testing.T) {
    // Create test candles
    candles := []marketdata.Candle{
        // Add test candle data here
    }

    // Create your strategy with test parameters
    cfg := YourStrategyConfig{
        Name:    "your-strategy-name",
        Version: "v1",
        Symbol:  "BTCUSDT",
        Interval: "1m",
        Param1:  9,
        Param2:  2.0,
    }
    strategy := NewYourStrategy(cfg)

    // Call Evaluate
    signal, err := strategy.Evaluate(candles)
    if err != nil {
        t.Errorf("Unexpected error: %v", err)
        return
    }

    // Validate the signal
    if signal.StrategyName != "your-strategy-name" {
        t.Errorf("Expected strategy name 'your-strategy-name', got '%s'", signal.StrategyName)
    }
    // Add more assertions as needed
}
```

### Integration Testing
To test your strategy in the context of the full system:
1. Set the appropriate environment variables:
   ```powershell
   $env:STRATEGY_NAME="your-strategy-name"
   $env:STRATEGY_SYMBOL="BTCUSDT"
   $env:STRATEGY_INTERVAL="1m"
   # Set your strategy-specific parameters
   $env:STRATEGY_FAST_PERIOD="9"
   $env:STRATEGY_SLOW_PERIOD="21"
   ```
2. Run the strategy command:
   ```powershell
   go run ./cmd/strategy
   ```

## Best Practices

### 1. Parameter Validation
Always validate strategy parameters in both the `Settings.Validate()` method and your strategy's `Evaluate` method. Provide clear error messages when validation fails.

### 2. Strength Normalization
When calculating signal strength, try to normalize it to a 0-100 scale (or higher for stronger signals) to ensure consistency across different strategies. This helps the risk module make consistent decisions.

### 3. Clear Reasoning
Provide clear, human-readable explanations in the `Reason` field. This makes debugging easier and helps users understand why signals were generated.

### 4. Proper Time Handling
Set the `GeneratedAt` field appropriately. For strategies that react to closed candles, use the close time of the most recent candle. For real-time strategies, use the current time.

### 5. Error Handling
Return meaningful errors from the `Evaluate` method when strategy execution fails (e.g., insufficient data, invalid parameters). The system will log these errors appropriately.

### 6. Performance Considerations
Since strategies are evaluated on every new candle, ensure your implementation is efficient. Avoid expensive computations in the critical path if possible.

### 7. Documentation
Comment your code clearly, explaining:
- What your strategy does
- How to interpret the parameters
- Any limitations or assumptions
- How the signal strength is calculated

### 8. Cost-Aware Strategy Requirements
Every executable strategy must be evaluated net of fees and slippage. A strategy should not be considered production-ready unless its average trade is greater than the estimated round-trip cost and it remains positive against a fair buy-and-hold benchmark.

For high-frequency intervals such as `1m`, use stricter cooldown and minimum holding bars. A useful first threshold is:

- average trade greater than round-trip cost
- profit factor greater than 1.2
- positive excess return
- trades per day below the template profile limit
- max drawdown inside the strategy's declared risk limit

## Example: Simple Moving Average Crossover

For reference, here's how the SMA crossover strategy works (simplified):

1. Calculates short-term and long-term simple moving averages
2. Generates a buy signal when the short-term SMA crosses above the long-term SMA
3. Generates a sell signal when the short-term SMA crosses below the long-term SMA
4. Uses a minimum gap threshold to filter out noise
5. Normalizes strength as the percentage difference between the two SMAs

This example demonstrates:
- Parameter validation (ensuring fast period < slow period)
- Sufficient data checking
- Clear signal generation logic
- Human-readable reason strings
- Normalized strength calculation
- Proper time handling

By following these patterns, you can create strategies that integrate seamlessly with the Sentra system.
