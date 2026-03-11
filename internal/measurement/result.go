package measurement

import "time"

// Stats summarizes latency and loss.
type Stats struct {
	Sent     int           `json:"sent"`
	Received int           `json:"received"`
	LossRate float64       `json:"loss_rate"`
	MinRTT   time.Duration `json:"min_rtt"`
	MaxRTT   time.Duration `json:"max_rtt"`
	AvgRTT   time.Duration `json:"avg_rtt"`
	Jitter   time.Duration `json:"jitter"`
	lastRTT  time.Duration `json:"-"` // for jitter computation only
}

// RoundResult holds statistics for a single round (Design.md §5, §8).
type RoundResult struct {
	Index int       `json:"index"`
	From  time.Time `json:"from"`
	To    time.Time `json:"to"`
	Stats Stats     `json:"stats"`
}

// WindowStats is used only for continuous mode sliding window.
type WindowStats struct {
	WindowSeconds int   `json:"window_seconds"`
	Stats         Stats `json:"stats"`
}

// Result is the unified measurement result structure (Design.md §8).
type Result struct {
	TaskID   TaskID    `json:"task_id"`
	NodeID   string    `json:"node_id"`
	Target   string    `json:"target"`
	Type     string    `json:"type"`
	Mode     Mode      `json:"mode"`
	Started  time.Time `json:"started"`
	Finished time.Time `json:"finished"`

	Total       Stats          `json:"total"`
	Rounds      []RoundResult  `json:"rounds,omitempty"`
	Window      *WindowStats   `json:"window,omitempty"`
	Status      string         `json:"status"` // running/finished/failed/stopped
	Error       string         `json:"error,omitempty"`
	LastUpdated time.Time      `json:"last_updated"`
}

