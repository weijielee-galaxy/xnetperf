package probe

import (
	"time"
)

// ProbeSum mary 探测汇总
type ProbeSummary struct {
	Timestamp      string        `json:"timestamp"`
	Results        []ProbeResult `json:"results"`
	RunningHosts   int           `json:"running_hosts"`
	CompletedHosts int           `json:"completed_hosts"`
	ErrorHosts     int           `json:"error_hosts"`
	TotalProcesses int           `json:"total_processes"`
	AllCompleted   bool          `json:"all_completed"`
}

func (p *Prober) DoProbeAndGetSummary() (*ProbeSummary, error) {

	// 探测所有主机
	results, err := p.DoProbe()
	if err != nil {
		return nil, err
	}

	// 统计信息
	summary := &ProbeSummary{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Results:   results,
	}

	for _, result := range results {
		switch result.Status {
		case "RUNNING":
			summary.RunningHosts++
			summary.TotalProcesses += result.ProcessCount
		case "COMPLETED":
			summary.CompletedHosts++
		case "ERROR":
			summary.ErrorHosts++
		}
	}

	summary.AllCompleted = (summary.CompletedHosts == len(results))

	return summary, nil
}
