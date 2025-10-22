package generator

import (
	"xnetperf/config"
	"xnetperf/internal/tools"
)

const delimiter = " && \\ \n"

type ScriptGeneratorIface interface {
	GenerateScripts() (*ScriptResult, error)

	// helper methods can be added here
	buildIbWriteBwCommand(device, port, targetIP string, cfg *config.ClientConfig) string
	buildIbWriteLatCommand(device, port, targetIP string, cfg *config.ClientConfig) string
}

type ScriptGenerator struct {
}

func (sg *ScriptGenerator) GenerateScripts() (*ScriptResult, error) {
	return nil, nil
}

func (sg *ScriptGenerator) buildIbWriteBwCommand(cfg *config.Config, hca string, port int, targetIP string, rFileName string) string {
	cmd := tools.NewIBWriteBwCommand().
		Device(hca).
		QueuePairs(cfg.QpNum).
		MessageSize(cfg.MessageSizeBytes).
		Port(port).
		RunInfinitely(cfg.Run.Infinitely).
		Duration(cfg.Run.DurationSeconds).
		RdmaCm(cfg.RdmaCm).
		GidIndex(cfg.GidIndex)
	if cfg.Report.Enable {
		cmd = cmd.EnableReport(rFileName)
	}
	if targetIP != "" {
		cmd = cmd.AsClient(targetIP)
	} else {
		cmd = cmd.AsServer()
	}
	return cmd.String()
}

func (sg *ScriptGenerator) buildIbWriteLatCommand(cfg *config.Config, hca string, port int, targetIP string, rFileName string) string {
	cmd := tools.NewIBWriteLatCommand().
		Device(hca).
		Port(port).
		RunInfinitely(false).
		Duration(5).
		RdmaCm(cfg.RdmaCm).
		GidIndex(cfg.GidIndex)

	if targetIP != "" {
		cmd = cmd.AsClient(targetIP)
	} else {
		cmd = cmd.AsServer()
	}

	if cfg.Report.Enable {
		cmd = cmd.EnableReport(rFileName)
	}
	return cmd.String()
}
