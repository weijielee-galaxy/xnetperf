package stream

import (
	"fmt"
	"os"
	"xnetperf/config"
)

func ClearStreamScriptDir(cfg *config.Config) {
	dir := cfg.OutputDir()
	err := os.RemoveAll(dir)
	if err != nil {
		fmt.Printf("Error clearing stream script directory: %v\n", err)
		return
	}
	err = os.Mkdir(dir, 0755)
	if err != nil {
		fmt.Printf("Error creating stream script directory: %v\n", err)
		return
	}
	fmt.Printf("Cleared stream script directory: %s\n", dir)
}
