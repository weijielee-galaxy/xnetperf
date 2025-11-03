package generator

import "strings"

type ScriptResult struct {
	ServerScripts []*HostScript
	ClientScripts []*HostScript
}

func BuildHostScriptsFromCmdMap(hostCmds map[string][]string, user string) []*HostScript {
	hs := []*HostScript{}
	for host, cmds := range hostCmds {
		cCmds := []string{}
		for _, cmd := range cmds {
			cCmds = append(cCmds, "( "+cmd+" )")
		}
		if user != "" {
			host = user + "@" + host
		}
		hs = append(hs, &HostScript{
			Host:         host,
			Command:      strings.Join(cCmds, delimiter),
			CommandCount: len(cmds),
		})
	}
	return hs
}

type HostScript struct {
	Host         string
	Command      string
	CommandCount int
}
