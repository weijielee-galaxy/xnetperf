package cmd

import (
	"fmt"
	"xnetperf/config"
	"xnetperf/server"

	"github.com/spf13/cobra"
)

var serverPort int

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start HTTP API server for configuration management",
	Long: `Start an HTTP API server that provides endpoints for managing configuration files.
Example:
  xnetperf server
  xnetperf server --port 8080`,
	Run: runServer,
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().IntVarP(&serverPort, "port", "p", 8080, "HTTP server port")
}

func runServer(cmd *cobra.Command, args []string) {
	// Ensure config.yaml exists in current directory
	if err := config.EnsureConfigFile("config.yaml"); err != nil {
		fmt.Printf("Warning: %v\n", err)
	}

	srv := server.NewServer(serverPort)
	if err := srv.Start(); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}
