package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/yanjiulab/lopa/internal/measurement"
)

func init() {
	taskCmd := &cobra.Command{
		Use:   "task",
		Short: "Manage measurement tasks via daemon",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks on daemon",
		RunE:  runTaskList,
	}

	showCmd := &cobra.Command{
		Use:   "show <task-id>",
		Short: "Show details of a task",
		Args:  cobra.ExactArgs(1),
		RunE:  runTaskShow,
	}

	stopCmd := &cobra.Command{
		Use:   "stop <task-id>",
		Short: "Stop a running task",
		Args:  cobra.ExactArgs(1),
		RunE:  runTaskStop,
	}

	deleteCmd := &cobra.Command{
		Use:   "delete <task-id>",
		Short: "Delete a task from daemon",
		Args:  cobra.ExactArgs(1),
		RunE:  runTaskDelete,
	}

	taskCmd.AddCommand(listCmd, showCmd, stopCmd, deleteCmd)
	rootCmd.AddCommand(taskCmd)
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

func baseURL() string {
	return strings.TrimRight(DaemonAddr(), "/")
}

func runTaskList(cmd *cobra.Command, args []string) error {
	client := newHTTPClient()
	url := baseURL() + "/api/v1/tasks"

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to list tasks from daemon %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("daemon returned status %s when listing tasks", resp.Status)
	}

	var results []measurement.Result
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return fmt.Errorf("failed to decode task list: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("no tasks on daemon")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE\tMODE\tTARGET\tSTATUS\tSENT\tRECEIVED\tLOSS(%)")
	for _, r := range results {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%d\t%.2f\n",
			r.TaskID, r.Type, r.Mode, r.Target, r.Status,
			r.Total.Sent, r.Total.Received, r.Total.LossRate*100)
	}
	_ = w.Flush()
	return nil
}

func runTaskShow(cmd *cobra.Command, args []string) error {
	id := args[0]
	client := newHTTPClient()
	url := fmt.Sprintf("%s/api/v1/tasks/%s", baseURL(), id)

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to query task %s: %w", id, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("task not found: %s", id)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("daemon returned status %s when querying task", resp.Status)
	}

	var res measurement.Result
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return fmt.Errorf("failed to decode task result: %w", err)
	}

	printResult(res)
	if res.Window != nil {
		fmt.Printf("Window: last %ds, sent=%d, recv=%d, loss=%.2f%%, avg=%s, jitter=%s\n",
			res.Window.WindowSeconds,
			res.Window.Stats.Sent,
			res.Window.Stats.Received,
			res.Window.Stats.LossRate*100,
			res.Window.Stats.AvgRTT,
			res.Window.Stats.Jitter,
		)
	}
	return nil
}

func runTaskStop(cmd *cobra.Command, args []string) error {
	id := args[0]
	client := newHTTPClient()
	url := fmt.Sprintf("%s/api/v1/tasks/%s/stop", baseURL(), id)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to stop task %s: %w", id, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("daemon returned status %s when stopping task", resp.Status)
	}
	fmt.Printf("stop signal sent for task %s\n", id)
	return nil
}

func runTaskDelete(cmd *cobra.Command, args []string) error {
	id := args[0]
	client := newHTTPClient()
	url := fmt.Sprintf("%s/api/v1/tasks/%s", baseURL(), id)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete task %s: %w", id, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("task not found: %s", id)
	}
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("daemon returned status %s when deleting task", resp.Status)
	}
	fmt.Printf("task %s deleted from daemon\n", id)
	return nil
}

