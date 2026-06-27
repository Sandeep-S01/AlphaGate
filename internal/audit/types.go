package audit

import "time"

type Event struct {
	ID          string    `json:"id"`
	EventType   string    `json:"event_type"`
	Actor       string    `json:"actor"`
	Summary     string    `json:"summary"`
	DetailsJSON string    `json:"details_json"`
	CreatedAt   time.Time `json:"created_at"`
}

type Query struct {
	EventType string
	Actor     string
	From      time.Time
	To        time.Time
	Limit     int
}
