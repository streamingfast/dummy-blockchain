package app

import (
	"fmt"
	"runtime/debug"
)

func VersionString(version string) string {
	gitCommit := "unknown"
	gitDate := "unknown"

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				gitCommit = setting.Value[:7]
			case "vcs.time":
				gitDate = setting.Value
			}
		}
	}

	return fmt.Sprintf(
		"%v (Commit: %s, Commit Date: %s)",
		version, gitCommit, gitDate,
	)
}
