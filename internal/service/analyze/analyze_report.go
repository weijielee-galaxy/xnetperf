package analyze

import (
	"fmt"
	"os"
	"xnetperf/config"
)

// ReportData 报告数据结构
type ReportData struct {
	StreamType             string                                   `json:"stream_type"`
	TheoreticalBWPerClient float64                                  `json:"theoretical_bw_per_client,omitempty"`
	TotalServerBW          float64                                  `json:"total_server_bw,omitempty"`
	ClientCount            int                                      `json:"client_count,omitempty"`
	ClientData             map[string]map[string]*ClientDeviceData  `json:"client_data,omitempty"`
	ServerData             map[string]map[string]*ServerDeviceData  `json:"server_data,omitempty"`
	P2PData                map[string]map[string]*P2PDeviceDataInfo `json:"p2p_data,omitempty"`
	P2PSummary             *P2PSummary                              `json:"p2p_summary,omitempty"`
}

// ClientDeviceData 客户端设备数据
type ClientDeviceData struct {
	Hostname      string  `json:"hostname"`
	Device        string  `json:"device"`
	ActualBW      float64 `json:"actual_bw"`
	TheoreticalBW float64 `json:"theoretical_bw"`
	Delta         float64 `json:"delta"`
	DeltaPercent  float64 `json:"delta_percent"`
	Status        string  `json:"status"` // OK, NOT OK
}

// ServerDeviceData 服务端设备数据
type ServerDeviceData struct {
	Hostname      string  `json:"hostname"`
	Device        string  `json:"device"`
	RxBW          float64 `json:"rx_bw"`
	TheoreticalBW float64 `json:"theoretical_bw"`
	Delta         float64 `json:"delta"`
	DeltaPercent  float64 `json:"delta_percent"`
	Status        string  `json:"status"` // OK, NOT OK
}

// P2PDeviceData P2P设备数据
type P2PDeviceDataInfo struct {
	Hostname string  `json:"hostname"`
	Device   string  `json:"device"`
	AvgSpeed float64 `json:"avg_speed"`
	Count    int     `json:"count"`
}

// P2PSummary P2P汇总数据
type P2PSummary struct {
	TotalPairs int     `json:"total_pairs"`
	AvgSpeed   float64 `json:"avg_speed"`
}

// GenerateReport 生成报告数据
func (a *Analyzer) GenerateReport() (*ReportData, error) {
	reportsDir := "reports"

	// 检查 reports 目录是否存在
	if _, err := os.Stat(reportsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("reports directory not found: %s", reportsDir)
	}
	cfg := a.cfg
	report := &ReportData{
		StreamType: string(cfg.StreamType),
	}

	switch cfg.StreamType {
	case config.P2P:
		// P2P 分析 TODO

	default:
		// FullMesh 和 InCast 分析
		clientData, serverData, err := collectReportData(reportsDir, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to collect report data: %v", err)
		}

		// 计算理论带宽
		report.TotalServerBW = calculateTotalServerBandwidth(serverData, cfg.Speed)
		report.ClientCount = calculateClientCount(clientData)
		if report.ClientCount > 0 {
			report.TheoreticalBWPerClient = report.TotalServerBW / float64(report.ClientCount)
		}

		// 转换为 API 响应格式
		report.ClientData = convertClientData(clientData, report.TheoreticalBWPerClient)
		report.ServerData = convertServerData(serverData, a.cfg.Speed)
	}

	return report, nil
}

func convertP2PData(p2pData map[string]map[string]*DeviceData) map[string]map[string]*P2PDeviceDataInfo {
	result := make(map[string]map[string]*P2PDeviceDataInfo)

	for hostname, devices := range p2pData {
		result[hostname] = make(map[string]*P2PDeviceDataInfo)
		for device, data := range devices {
			result[hostname][device] = &P2PDeviceDataInfo{
				Hostname: hostname,
				Device:   device,
				AvgSpeed: data.BWSum / float64(data.Count),
				Count:    data.Count,
			}
		}
	}

	return result
}

func calculateP2PSummary(p2pData map[string]map[string]*DeviceData) *P2PSummary {
	totalPairs := 0
	totalSpeed := 0.0

	for _, devices := range p2pData {
		for _, data := range devices {
			totalPairs++
			totalSpeed += data.BWSum / float64(data.Count)
		}
	}

	summary := &P2PSummary{
		TotalPairs: totalPairs,
	}

	if totalPairs > 0 {
		summary.AvgSpeed = totalSpeed / float64(totalPairs)
	}

	return summary
}

func convertClientData(clientData map[string]map[string]*DeviceData, theoreticalBW float64) map[string]map[string]*ClientDeviceData {
	result := make(map[string]map[string]*ClientDeviceData)

	for hostname, devices := range clientData {
		result[hostname] = make(map[string]*ClientDeviceData)
		for device, data := range devices {
			actualBW := data.BWSum
			delta := actualBW - theoreticalBW
			deltaPercent := float64(0)
			if theoreticalBW > 0 {
				deltaPercent = (delta / theoreticalBW) * 100
			}

			status := "OK"
			if abs(deltaPercent) > 20 {
				status = "NOT OK"
			}

			result[hostname][device] = &ClientDeviceData{
				Hostname:      hostname,
				Device:        device,
				ActualBW:      actualBW,
				TheoreticalBW: theoreticalBW,
				Delta:         delta,
				DeltaPercent:  deltaPercent,
				Status:        status,
			}
		}
	}

	return result
}

func convertServerData(serverData map[string]map[string]*DeviceData, theoreticalBW float64) map[string]map[string]*ServerDeviceData {
	result := make(map[string]map[string]*ServerDeviceData)

	for hostname, devices := range serverData {
		result[hostname] = make(map[string]*ServerDeviceData)
		for device, data := range devices {

			delta := data.BWSum - theoreticalBW
			deltaPercent := float64(0)
			if theoreticalBW > 0 {
				deltaPercent = (delta / theoreticalBW) * 100
			}
			status := "OK"
			if abs(deltaPercent) > 20 {
				status = "NOT OK"
			}

			result[hostname][device] = &ServerDeviceData{
				Hostname:      hostname,
				Device:        device,
				RxBW:          data.BWSum,
				TheoreticalBW: theoreticalBW,
				Delta:         delta,
				DeltaPercent:  deltaPercent,
				Status:        status,
			}
		}
	}

	return result
}
