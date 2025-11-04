package cmd

import (
	"log"
	"xnetperf/config"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "xnetperf",
	Short: "xnetperf network test tool",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Load configuration
		var err error
		cfg, err = config.LoadConfig(cfgFile)
		if err != nil {
			log.Fatal(err)
		}
		// Initialize logger with global flags
		// logger.Init(logger.Config{
		// 	Level:  logger.LogLevel(logLevel),
		// 	Format: logFormat,
		// })
	},
}

func Execute() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "./config.yaml", "config file")
	rootCmd.AddCommand(precheckCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(collectCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(probeCmd)
	rootCmd.AddCommand(executeCmd)
	rootCmd.AddCommand(latCmd)
	_ = rootCmd.Execute()
}
