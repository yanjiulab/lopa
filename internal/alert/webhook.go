package alert

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/yanjiulab/lopa/internal/logger"
)

// EventAlert and EventRecovery are the values for Payload.Event.
const (
	EventAlert   = "alert"   // threshold exceeded
	EventRecovery = "recovery" // metrics returned to normal after alert
)

// Payload is the JSON body sent to the webhook URL (alert or recovery).
type Payload struct {
	Event         string  `json:"event"`          // "alert" or "recovery"
	TaskID        string  `json:"task_id"`
	NodeID       string  `json:"node_id"`
	Target       string  `json:"target"`
	Type         string  `json:"type"`
	Mode         string  `json:"mode"`
	Reason       string  `json:"reason"`         // "loss", "latency", or "recovery"
	Threshold    string  `json:"threshold"`      // human-readable threshold (alert only)
	CurrentValue string  `json:"current_value"`  // human-readable current value
	RecoveredFrom string  `json:"recovered_from,omitempty"` // for recovery: e.g. "loss, latency"
	WindowSeconds int    `json:"window_seconds"`
	Sent         int     `json:"sent"`
	Received     int     `json:"received"`
	LossRate     float64 `json:"loss_rate"`
	MinRTT       string  `json:"min_rtt"`
	MaxRTT       string  `json:"max_rtt"`
	AvgRTT       string  `json:"avg_rtt"`
	Timestamp    string  `json:"timestamp"`
}

// Notify sends the payload to the given URL via HTTP POST (async, non-blocking).
func Notify(url string, p Payload) {
	if url == "" {
		return
	}
	go func() {
		body, err := json.Marshal(p)
		if err != nil {
			logger.S().Warnw("alert payload marshal failed", "err", err)
			return
		}
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			logger.S().Warnw("alert request build failed", "err", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			logger.S().Warnw("alert webhook request failed", "url", url, "err", err)
			return
		}
		resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			logger.S().Warnw("alert webhook returned non-2xx", "url", url, "status", resp.StatusCode)
			return
		}
		logger.S().Infow("webhook sent", "url", url, "task_id", p.TaskID, "event", p.Event, "reason", p.Reason)
	}()
}
