package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yanjiulab/lopa/internal/measurement"
)

func init() {
	udpCmd := &cobra.Command{
		Use:   "udp [target]",
		Short: "UDP probe measurement (requires reflector at target)",
		Long:  "Target is reflector host:port (default port 8081). Sends UDP packets, measures RTT from echo.",
		Args:  cobra.ExactArgs(1),
		RunE:  runUdp,
	}

	udpCmd.Flags().String("mode", string(measurement.ModeCount), "mode: count|duration|continuous")
	udpCmd.Flags().Int("count", 4, "number of probes for count mode")
	udpCmd.Flags().Duration("duration", 10*time.Second, "duration for duration or continuous mode")
	udpCmd.Flags().Duration("interval", time.Second, "interval between probes")
	udpCmd.Flags().Duration("timeout", 3*time.Second, "timeout per probe")
	udpCmd.Flags().Int("size", 64, "UDP payload size in bytes (min 8)")
	udpCmd.Flags().String("ip-version", "", "ip version: ipv4|ipv6 (auto if empty)")
	udpCmd.Flags().String("source-ip", "", "source IP to bind")
	udpCmd.Flags().String("interface", "", "network interface name to bind")
	udpCmd.Flags().Int("rounds", 1, "rounds for count mode")
	udpCmd.Flags().Duration("round-interval", 0, "interval between rounds")
	udpCmd.Flags().Float64("loss-threshold", 0, "alert when window loss rate >= this (continuous mode)")
	udpCmd.Flags().Duration("latency-threshold", 0, "alert when window avg RTT >= this (continuous mode)")
	udpCmd.Flags().String("alert-callback-url", "", "webhook URL when threshold exceeded (continuous mode)")
	udpCmd.Flags().Int("port", 8081, "default reflector port when target has no port")

	rootCmd.AddCommand(udpCmd)
}

func runUdp(cmd *cobra.Command, args []string) error {
	target := args[0]
	if _, _, err := parseHostPort(target); err != nil {
		port, _ := cmd.Flags().GetInt("port")
		target = fmt.Sprintf("%s:%d", target, port)
	}

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
		Type:             "udp",
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
	createURL := base + "/api/v1/tasks/udp"
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
	fmt.Printf("started udp task %s to %s via daemon %s\n", id, target, base)

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
