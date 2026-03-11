package cli

import (
	"fmt"
	"net/http"
	"bytes"
	"encoding/json"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yanjiulab/lopa/internal/measurement"
)

func init() {
	pingCmd := &cobra.Command{
		Use:   "ping [target]",
		Short: "ICMP ping measurement",
		Args:  cobra.ExactArgs(1),
		RunE:  runPing,
	}

	pingCmd.Flags().String("mode", string(measurement.ModeCount), "mode: count|duration|continuous")
	pingCmd.Flags().Int("count", 4, "number of packets for count mode")
	pingCmd.Flags().Duration("duration", 10*time.Second, "duration for duration or continuous mode")
	pingCmd.Flags().Duration("interval", time.Second, "interval between packets")
	pingCmd.Flags().Duration("timeout", 3*time.Second, "timeout per packet")
	pingCmd.Flags().Int("size", 56, "ICMP payload size in bytes")
	pingCmd.Flags().String("ip-version", "", "ip version: ipv4|ipv6 (auto if empty)")
	pingCmd.Flags().String("source-ip", "", "source IP to bind")
	pingCmd.Flags().String("interface", "", "network interface name to bind")
	pingCmd.Flags().Int("rounds", 1, "rounds for count mode")
	pingCmd.Flags().Duration("round-interval", 0, "interval between rounds")
	// continuous mode: webhook alert thresholds
	pingCmd.Flags().Float64("loss-threshold", 0, "alert when window loss rate >= this (0=disabled, e.g. 0.05 for 5%%)")
	pingCmd.Flags().Duration("latency-threshold", 0, "alert when window avg RTT >= this (0=disabled)")
	pingCmd.Flags().String("alert-callback-url", "", "webhook URL to POST when threshold is exceeded (continuous mode)")

	rootCmd.AddCommand(pingCmd)
}

func runPing(cmd *cobra.Command, args []string) error {
	target := args[0]

	modeStr, _ := cmd.Flags().GetString("mode")
	count, _ := cmd.Flags().GetInt("count")
	duration, _ := cmd.Flags().GetDuration("duration")
	interval, _ := cmd.Flags().GetDuration("interval")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	size, _ := cmd.Flags().GetInt("size")
	ipVersion, _ := cmd.Flags().GetString("ip-version")
	sourceIP, _ := cmd.Flags().GetString("source-ip")
	iface, _ := cmd.Flags().GetString("interface")
	rounds, _ := cmd.Flags().GetInt("rounds")
	roundInterval, _ := cmd.Flags().GetDuration("round-interval")
	lossThreshold, _ := cmd.Flags().GetFloat64("loss-threshold")
	latencyThreshold, _ := cmd.Flags().GetDuration("latency-threshold")
	alertCallbackURL, _ := cmd.Flags().GetString("alert-callback-url")

	mode := measurement.Mode(modeStr)

	params := measurement.TaskParams{
		Type:             "ping",
		Target:           target,
		IPVersion:        ipVersion,
		SourceIP:         sourceIP,
		Interface:        iface,
		Interval:         interval,
		Timeout:          timeout,
		PacketSize:       size,
		Mode:             mode,
		Count:            count,
		Duration:         duration,
		Rounds:           rounds,
		RoundDelay:       roundInterval,
		LossThreshold:    lossThreshold,
		LatencyThreshold: latencyThreshold,
		AlertCallbackURL: alertCallbackURL,
	}

	base := strings.TrimRight(DaemonAddr(), "/")
	client := &http.Client{Timeout: 10 * time.Second}

	// Create task via daemon HTTP API
	createURL := base + "/api/v1/tasks/ping"
	body, err := json.Marshal(params)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", createURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to contact daemon at %s: %w", createURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("daemon returned status %s when creating task", resp.Status)
	}

	var createResp struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return fmt.Errorf("failed to decode daemon response: %w", err)
	}
	if createResp.ID == "" {
		return fmt.Errorf("daemon returned empty task id")
	}

	id := measurement.TaskID(createResp.ID)
	fmt.Printf("started ping task %s to %s via daemon %s\n", id, target, base)

	// For count/duration: wait until finished and print final result once.
	// For continuous: print window stats periodically.
	switch mode {
	case measurement.ModeCount, measurement.ModeDuration:
		for {
			getURL := fmt.Sprintf("%s/api/v1/tasks/%s", base, id)
			resp, err := client.Get(getURL)
			if err != nil {
				return fmt.Errorf("failed to query task %s: %w", id, err)
			}
			if resp.StatusCode == http.StatusNotFound {
				resp.Body.Close()
				return fmt.Errorf("task not found: %s", id)
			}
			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				return fmt.Errorf("daemon returned status %s when querying task", resp.Status)
			}
			var res measurement.Result
			if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
				resp.Body.Close()
				return fmt.Errorf("failed to decode task result: %w", err)
			}
			resp.Body.Close()
			if res.Status == "finished" || res.Status == "failed" {
				printResult(res)
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
	case measurement.ModeContinuous:
		for {
			getURL := fmt.Sprintf("%s/api/v1/tasks/%s", base, id)
			resp, err := client.Get(getURL)
			if err != nil {
				return fmt.Errorf("failed to query task %s: %w", id, err)
			}
			if resp.StatusCode == http.StatusNotFound {
				resp.Body.Close()
				return fmt.Errorf("task not found: %s", id)
			}
			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				return fmt.Errorf("daemon returned status %s when querying task", resp.Status)
			}
			var res measurement.Result
			if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
				resp.Body.Close()
				return fmt.Errorf("failed to decode task result: %w", err)
			}
			resp.Body.Close()
			if res.Window != nil {
				fmt.Printf("task=%s target=%s window=%ds sent=%d recv=%d loss=%.2f avg=%s\n",
					res.TaskID, res.Target, res.Window.WindowSeconds,
					res.Window.Stats.Sent, res.Window.Stats.Received,
					res.Window.Stats.LossRate*100, res.Window.Stats.AvgRTT)
			}
			time.Sleep(2 * time.Second)
		}
	default:
		return fmt.Errorf("unknown mode: %s", mode)
	}

	return nil
}

func printResult(res measurement.Result) {
	fmt.Printf("Task %s (%s, mode=%s) finished\n", res.TaskID, res.Type, res.Mode)
	fmt.Printf("Target: %s\n", res.Target)
	fmt.Printf("Sent: %d, Received: %d, Loss: %.2f%%\n",
		res.Total.Sent, res.Total.Received, res.Total.LossRate*100)
	fmt.Printf("RTT min/avg/max/jitter: %s/%s/%s/%s\n",
		res.Total.MinRTT, res.Total.AvgRTT, res.Total.MaxRTT, res.Total.Jitter)
	if len(res.Rounds) > 1 {
		fmt.Printf("Rounds: %d\n", len(res.Rounds))
		for _, rr := range res.Rounds {
			fmt.Printf("  Round %d: sent=%d recv=%d loss=%.2f%% min/avg/max/jitter=%s/%s/%s/%s\n",
				rr.Index,
				rr.Stats.Sent,
				rr.Stats.Received,
				rr.Stats.LossRate*100,
				rr.Stats.MinRTT,
				rr.Stats.AvgRTT,
				rr.Stats.MaxRTT,
				rr.Stats.Jitter,
			)
		}
	}
}

