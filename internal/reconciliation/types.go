package reconciliation

import "time"

type Status string

const (
	StatusMatched  Status = "matched"
	StatusMismatch Status = "mismatch"
	StatusFailed   Status = "failed"
)

type MismatchKind string

const (
	MismatchBalance  MismatchKind = "balance"
	MismatchPosition MismatchKind = "position"
	MismatchOrder    MismatchKind = "order"
)

type Balance struct {
	Asset string  `json:"asset"`
	Free  float64 `json:"free"`
}

type Position struct {
	Symbol        string  `json:"symbol"`
	Quantity      float64 `json:"quantity"`
	QuoteExposure float64 `json:"quote_exposure"`
}

type Order struct {
	ClientOrderID string `json:"client_order_id"`
	Symbol        string `json:"symbol"`
	Status        string `json:"status"`
}

type Snapshot struct {
	Balances  []Balance  `json:"balances"`
	Positions []Position `json:"positions"`
	Orders    []Order    `json:"orders"`
}

type Mismatch struct {
	Kind          MismatchKind `json:"kind"`
	Key           string       `json:"key"`
	InternalValue string       `json:"internal_value"`
	ExternalValue string       `json:"external_value"`
	Severity      string       `json:"severity"`
}

type Run struct {
	ID         string     `json:"id"`
	Status     Status     `json:"status"`
	Mismatches []Mismatch `json:"mismatches"`
	CreatedAt  time.Time  `json:"created_at"`
}

type Query struct {
	Limit int
}
