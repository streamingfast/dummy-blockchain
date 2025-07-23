package app

import (
	"fmt"
	"runtime/debug"
)

var (
	Version = "0.0.1"
)

func VersionString() string {
	gitCommit := "unknown"
	gitDate := "unknown"

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				gitCommit = setting.Value
			case "vcs.time":
				gitDate = setting.Value
			}
		}
	}

	return fmt.Sprintf(
		"%v (Commit=%q Commit Date=%q)",
		Version, gitCommit, gitDate,
	)
}
