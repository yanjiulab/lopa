package measurement

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yanjiulab/lopa/internal/alert"
	"github.com/yanjiulab/lopa/internal/logger"
	"github.com/yanjiulab/lopa/internal/protocol"
)

func (e *Engine) runPingCount(ctx context.Context, p protocol.Pinger, task *Task, result *Result) {
	log := logger.S()
	params := task.Params
	if params.Count <= 0 {
		params.Count = 4
	}
	if params.Interval <= 0 {
		params.Interval = time.Second
	}

	var rounds int = params.Rounds
	if rounds <= 0 {
		rounds = 1
	}

	for rIdx := 0; rIdx < rounds; rIdx++ {
		select {
		case <-ctx.Done():
			e.updateResult(task.ID, func(res *Result) {
				res.Status = "stopped"
				res.Finished = time.Now()
			})
			return
		default:
		}

		roundStats := Stats{}
		roundStart := time.Now()

		for i := 0; i < params.Count; i++ {
			select {
			case <-ctx.Done():
				e.updateResult(task.ID, func(res *Result) {
					res.Status = "stopped"
					res.Finished = time.Now()
				})
				return
			default:
			}

			rtt, ok, err := p.Ping(ctx)
			if err != nil {
				log.Warnf("ping error task=%s: %v", task.ID, err)
			}

			computeStats(&roundStats, rtt, ok)
			e.updateResult(task.ID, func(res *Result) {
				computeStats(&res.Total, rtt, ok)
			})

			e.updateResult(task.ID, func(res *Result) {
				res.Status = "running"
			})

			time.Sleep(params.Interval)
		}

		roundEnd := time.Now()
		e.updateResult(task.ID, func(res *Result) {
			res.Rounds = append(res.Rounds, RoundResult{
				Index: rIdx + 1,
				From:  roundStart,
				To:    roundEnd,
				Stats: roundStats,
			})
		})

		if rIdx < rounds-1 && params.RoundDelay > 0 {
			time.Sleep(params.RoundDelay)
		}
	}

	e.updateResult(task.ID, func(res *Result) {
		res.Status = "finished"
		res.Finished = time.Now()
	})
	log.Infof("ping count task finished: %s", task.ID)
}

func (e *Engine) runPingDuration(ctx context.Context, p protocol.Pinger, task *Task, result *Result) {
	log := logger.S()
	params := task.Params
	if params.Duration <= 0 {
		params.Duration = 10 * time.Second
	}
	if params.Interval <= 0 {
		params.Interval = time.Second
	}

	endTime := time.Now().Add(params.Duration)
	stats := Stats{}
	roundStart := time.Now()

	for time.Now().Before(endTime) {
		select {
		case <-ctx.Done():
			e.updateResult(task.ID, func(res *Result) {
				res.Status = "stopped"
				res.Finished = time.Now()
			})
			return
		default:
		}

		rtt, ok, err := p.Ping(ctx)
		if err != nil {
			log.Warnf("ping error task=%s: %v", task.ID, err)
		}
		computeStats(&stats, rtt, ok)
		e.updateResult(task.ID, func(res *Result) {
			computeStats(&res.Total, rtt, ok)
			res.Status = "running"
		})

		time.Sleep(params.Interval)
	}

	roundEnd := time.Now()
	e.updateResult(task.ID, func(res *Result) {
		res.Rounds = append(res.Rounds, RoundResult{
			Index: 1,
			From:  roundStart,
			To:    roundEnd,
			Stats: stats,
		})
		res.Status = "finished"
		res.Finished = time.Now()
	})

	log.Infof("ping duration task finished: %s", task.ID)
}

// webhookCooldown prevents flooding the callback; only one alert per task per this interval.
const webhookCooldown = 60 * time.Second

func (e *Engine) runPingContinuous(ctx context.Context, p protocol.Pinger, task *Task, result *Result) {
	log := logger.S()
	params := task.Params
	if params.Interval <= 0 {
		params.Interval = time.Second
	}

	// sliding window: last N seconds, default 60
	windowSeconds := 60
	if params.Duration > 0 {
		windowSeconds = int(params.Duration.Seconds())
	}
	type sample struct {
		t   time.Time
		rtt time.Duration
		ok  bool
	}
	var samples []sample
	var lastWebhookAt time.Time
	var alertingLoss, alertingLatency bool

	ticker := time.NewTicker(params.Interval)
	defer ticker.Stop()

	makePayload := func(ts time.Time, wstats Stats, event, reason, threshold, currentValue, recoveredFrom string) alert.Payload {
		return alert.Payload{
			Event:         event,
			TaskID:        string(task.ID),
			NodeID:        task.NodeID,
			Target:        params.Target,
			Type:          params.Type,
			Mode:          string(params.Mode),
			Reason:        reason,
			Threshold:     threshold,
			CurrentValue:  currentValue,
			RecoveredFrom: recoveredFrom,
			WindowSeconds: windowSeconds,
			Sent:          wstats.Sent,
			Received:      wstats.Received,
			LossRate:      wstats.LossRate,
			MinRTT:        wstats.MinRTT.String(),
			MaxRTT:        wstats.MaxRTT.String(),
			AvgRTT:        wstats.AvgRTT.String(),
			Timestamp:     ts.UTC().Format(time.RFC3339),
		}
	}

	for {
		select {
		case <-ctx.Done():
			e.updateResult(task.ID, func(res *Result) {
				res.Status = "stopped"
				res.Finished = time.Now()
			})
			log.Infof("ping continuous task stopped: %s", task.ID)
			return
		case <-ticker.C:
			rtt, ok, err := p.Ping(ctx)
			if err != nil {
				log.Warnf("ping error task=%s: %v", task.ID, err)
			}

			now := time.Now()
			samples = append(samples, sample{t: now, rtt: rtt, ok: ok})

			// prune old samples
			cutoff := now.Add(-time.Duration(windowSeconds) * time.Second)
			n := 0
			for _, s := range samples {
				if s.t.After(cutoff) {
					samples[n] = s
					n++
				}
			}
			samples = samples[:n]

			// recompute window stats
			wstats := Stats{}
			for _, s := range samples {
				computeStats(&wstats, s.rtt, s.ok)
			}

			e.updateResult(task.ID, func(res *Result) {
				computeStats(&res.Total, rtt, ok)
				res.Status = "running"
				res.Window = &WindowStats{
					WindowSeconds: windowSeconds,
					Stats:         wstats,
				}
			})

			if params.AlertCallbackURL == "" {
				continue
			}

			lossExceeded := params.LossThreshold > 0 && wstats.LossRate >= params.LossThreshold
			latencyExceeded := params.LatencyThreshold > 0 && wstats.AvgRTT >= params.LatencyThreshold
			normal := !lossExceeded && !latencyExceeded

			// Recovery: was alerting and now both metrics are normal -> send recovery immediately
			if normal && (alertingLoss || alertingLatency) {
				var recoveredFrom []string
				if alertingLoss {
					recoveredFrom = append(recoveredFrom, "loss")
				}
				if alertingLatency {
					recoveredFrom = append(recoveredFrom, "latency")
				}
				alertingLoss = false
				alertingLatency = false
				cv := fmt.Sprintf("loss=%.2f%%, avg_rtt=%s", wstats.LossRate*100, wstats.AvgRTT)
				alert.Notify(params.AlertCallbackURL, makePayload(now, wstats, alert.EventRecovery, "recovery", "", cv, strings.Join(recoveredFrom, ", ")))
				continue
			}

			// Alert: exceed threshold, cooldown elapsed
			if time.Since(lastWebhookAt) < webhookCooldown {
				if lossExceeded {
					alertingLoss = true
				}
				if latencyExceeded {
					alertingLatency = true
				}
				continue
			}

			var reason, threshold, currentValue string
			if lossExceeded {
				reason = "loss"
				threshold = fmt.Sprintf("%.2f%%", params.LossThreshold*100)
				currentValue = fmt.Sprintf("%.2f%%", wstats.LossRate*100)
				alertingLoss = true
			} else if latencyExceeded {
				reason = "latency"
				threshold = params.LatencyThreshold.String()
				currentValue = wstats.AvgRTT.String()
				alertingLatency = true
			} else {
				continue
			}

			lastWebhookAt = now
			alert.Notify(params.AlertCallbackURL, makePayload(now, wstats, alert.EventAlert, reason, threshold, currentValue, ""))
		}
	}
}

