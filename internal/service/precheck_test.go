package service

import (
	"fmt"
	"testing"
)

func TestPrecheckHca(t *testing.T) {
	hca := "mlx5_0"
	combinedCmd := fmt.Sprintf(`
		echo "PHYS_STATE:$(cat /sys/class/infiniband/%s/ports/1/phys_state 2>/dev/null || echo 'ERROR')" && \
		echo "STATE:$(cat /sys/class/infiniband/%s/ports/1/state 2>/dev/null || echo 'ERROR')" && \
		echo "SPEED:$(cat /sys/class/infiniband/%s/ports/1/rate 2>/dev/null || echo 'ERROR')" && \
		echo "FW_VER:$(cat /sys/class/infiniband/%s/fw_ver 2>/dev/null || echo 'ERROR')" && \
		echo "BOARD_ID:$(cat /sys/class/infiniband/%s/board_id 2>/dev/null || echo 'ERROR')" && \
		echo "SERIAL:$(cat /sys/class/dmi/id/product_serial 2>/dev/null || echo 'ERROR')"
	`, hca, hca, hca, hca, hca)

	fmt.Println(combinedCmd)
}

func TestBuildHostCommands(t *testing.T) {
	hostHCAs := map[string][]string{
		"cetus-g88-002": {"mlx5_0", "mlx5_1"},
		"host2":         {"mlx5_0"},
		"host3":         {"mlx5_2", "mlx5_3", "mlx5_4"},
	}
	commands := buildHostCommands(hostHCAs)

	for host, cmd := range commands {
		fmt.Printf("Host: %s, Command: %s\n", host, cmd)
	}
}

func TestColorOutput(t *testing.T) {
	fmt.Println("Hello, World!")
	fmt.Println(ColorGreen + "This is a green text!" + ColorReset)
}
