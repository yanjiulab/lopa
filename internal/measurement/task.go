package measurement

import "time"

// Mode represents measurement mode: count, duration, continuous.
type Mode string

const (
	ModeCount      Mode = "count"
	ModeDuration   Mode = "duration"
	ModeContinuous Mode = "continuous"
)

// TaskParams describes the unified measurement parameters (Design.md §7).
type TaskParams struct {
	// Basic
	Type   string `json:"type"`   // ping/tcp/udp/twamp
	Target string `json:"target"` // ip/hostname:port if needed

	IPVersion string `json:"ip_version"` // ipv4/ipv6

	Interval   time.Duration `json:"interval"`
	Timeout    time.Duration `json:"timeout"`
	PacketSize int           `json:"packet_size"`

	SourceIP   string `json:"source_ip"`
	Interface  string `json:"interface"`
	Mode       Mode   `json:"mode"`
	Count      int    `json:"count"`    // for count mode
	Duration   time.Duration `json:"duration"` // for duration mode
	Rounds     int           `json:"rounds"`
	RoundDelay time.Duration `json:"round_delay"`

	// Continuous only
	LossThreshold     float64       `json:"loss_threshold"`
	LatencyThreshold  time.Duration `json:"latency_threshold"`
	AlertCallbackURL  string        `json:"alert_callback_url"`
}

// TaskID is a globally unique identifier: <node-uid>-<seq>.
type TaskID string

// Task represents a running or finished measurement task.
type Task struct {
	ID        TaskID     `json:"id"`
	Params    TaskParams `json:"params"`
	NodeID    string     `json:"node_id"`
	CreatedAt time.Time  `json:"created_at"`
}

