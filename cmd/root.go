package cmd

import (
	"log"
	"xnetperf/config"
	"xnetperf/pkg/tools/logger"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
)

func GetConfig() *config.Config {
	return cfg
}

var rootCmd = &cobra.Command{
	Use:   "xnetperf",
	Short: "xnetperf network test tool",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// 1. Load configuration
		var err error
		cfg, err = config.LoadConfig(cfgFile)
		if err != nil {
			log.Fatal(err)
		}

		// 2. Apply defaults to ensure logger config is set
		// 由于部分配置逻辑和默认值不一致，所以先注释掉
		// cfg.ApplyDefaults()

		// 3. Validate logger config
		if err := cfg.Logger.ValidateLogLevel(); err != nil {
			log.Fatalf("Invalid logger configuration: %v", err)
		}
		if err := cfg.Logger.ValidateLogFormat(); err != nil {
			log.Fatalf("Invalid logger configuration: %v", err)
		}

		// 4. Initialize logger with config values
		logger.Init(logger.Config{
			Level:  logger.LogLevel(cfg.Logger.LogLevel),
			Format: cfg.Logger.LogFormat,
		})

		// 5. Log successful initialization
		logger.Info("✓ Config loaded", "file", cfgFile)
		logger.Debug("Logger initialized", "level", cfg.Logger.LogLevel, "format", cfg.Logger.LogFormat)
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
	rootCmd.AddCommand(checkConnCmd)
	_ = rootCmd.Execute()
}
