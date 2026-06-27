package pine

import "time"

// PineStrategy represents a user-created Pine Script strategy stored in the database.
type PineStrategy struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	PineCode        string    `json:"pine_code"`
	ConvertedConfig IRConfig  `json:"converted_config"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// IRConfig is the Intermediate Representation produced by parsing Pine Script.
// It encodes all indicators, conditions, and execution rules declaratively.
type IRConfig struct {
	Indicators map[string]IndicatorDef `json:"indicators"`
	Conditions map[string]Expression   `json:"conditions"`
	Rules      []ExecutionRule         `json:"rules"`
}

// IndicatorDef declares a technical indicator.
// Type is one of: "sma", "ema", "rsi", "atr", "macd", "bb".
// Source is the data feed: "close", "open", "high", "low", "volume", or another indicator variable.
// Params are period values or multipliers depending on the indicator type.
type IndicatorDef struct {
	Type   string    `json:"type"`
	Source string    `json:"source"`
	Params []float64 `json:"params"`
}

// Expression is a recursive AST node for logical conditions.
// Op is one of: ">", "<", ">=", "<=", "==", "!=", "crossover", "crossunder", "and", "or", "not", "ref".
// Args hold sub-expressions. Val holds a literal number or variable reference when Op is "ref".
type Expression struct {
	Op   string       `json:"op"`
	Args []Expression `json:"args,omitempty"`
	Val  string       `json:"val,omitempty"`
}

// ExecutionRule maps a named condition to a strategy action.
// Action is "entry" or "close". Direction is "long" or "short".
type ExecutionRule struct {
	Condition string `json:"condition"`
	Action    string `json:"action"`
	ID        string `json:"id"`
	Direction string `json:"direction"`
}

// ParseResult is returned by the parser. It contains the compiled IR, any
// warnings (non-fatal issues), and any errors (fatal issues that prevent compilation).
type ParseResult struct {
	Config   IRConfig `json:"config"`
	Warnings []string `json:"warnings,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}

// ValidationResult is returned by the validate endpoint.
type ValidationResult struct {
	Valid      bool              `json:"valid"`
	Config     IRConfig          `json:"config"`
	Warnings   []string          `json:"warnings,omitempty"`
	Errors     []string          `json:"errors,omitempty"`
	Indicators []IndicatorInfo   `json:"indicators,omitempty"`
	Rules      []ExecutionRule   `json:"rules,omitempty"`
}

// IndicatorInfo provides a human-readable description of a detected indicator.
type IndicatorInfo struct {
	Name   string    `json:"name"`
	Type   string    `json:"type"`
	Source string    `json:"source"`
	Params []float64 `json:"params"`
}
