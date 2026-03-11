package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "lopa",
	Short: "Lopa is a lightweight network measurement tool",
	Long:  "Lopa is a lightweight, centralized, single-binary network quality measurement and monitoring tool.",
}

var daemonAddr string

// DaemonAddr returns the configured lopa daemon base URL.
func DaemonAddr() string {
	return daemonAddr
}

// Execute is the entrypoint for the CLI.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize()

	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	rootCmd.PersistentFlags().BoolP("help", "h", false, "show help for lopa")

	defaultDaemon := os.Getenv("LOPA_DAEMON_ADDR")
	if defaultDaemon == "" {
		defaultDaemon = "http://127.0.0.1:8080"
	}
	rootCmd.PersistentFlags().StringVar(&daemonAddr, "daemon", defaultDaemon, "lopa daemon base URL")

	// Ensure correct exit codes.
	rootCmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		if err != nil {
			c.Println("Error:", err)
			os.Exit(2)
		}
		return nil
	})
}

