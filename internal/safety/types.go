package safety

import "time"

type Status struct {
	KillSwitchActive bool      `json:"kill_switch_active"`
	Reason           string    `json:"reason"`
	UpdatedBy        string    `json:"updated_by"`
	UpdatedAt        time.Time `json:"updated_at"`
}
