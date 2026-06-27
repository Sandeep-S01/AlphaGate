package execution

type Status struct {
	Mode               string `json:"mode"`
	PaperEnabled       bool   `json:"paper_enabled"`
	ExchangeAdapter    string `json:"exchange_adapter"`
	LiveTradingEnabled bool   `json:"live_trading_enabled"`
	RetryAttempts      int    `json:"retry_attempts"`
	Timeout            string `json:"timeout"`
	LastError          string `json:"last_error"`
}
