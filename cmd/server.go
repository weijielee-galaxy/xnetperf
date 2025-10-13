package cmd

import (
	"fmt"

	"xnetperf/server"

	"github.com/spf13/cobra"
)

var serverPort int

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start HTTP API server for configuration management",
	Long: `Start an HTTP API server that provides endpoints for managing configuration files.

The server provides the following APIs:
  - GET    /api/configs        List all configuration files
  - GET    /api/configs/:name  Get a specific configuration file
  - POST   /api/configs        Create a new configuration file
  - PUT    /api/configs/:name  Update a configuration file
  - DELETE /api/configs/:name  Delete a configuration file (except default config.yaml)

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
	srv := server.NewServer(serverPort)
	if err := srv.Start(); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}
