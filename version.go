package orchestrator_graph

import "runtime/debug"

func Version() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		if bi.Main.Path == "github.com/stellwerk-labs/platform-orchestrator-graph" {
			return bi.Main.Version
		}
		for _, dep := range bi.Deps {
			if dep.Path == "github.com/stellwerk-labs/platform-orchestrator-graph" {
				return dep.Version
			}
		}
	}
	return ""
}
