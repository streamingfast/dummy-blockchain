package main

import (
	"fmt"
)

var (
	Version     = "0.0.1"
	BuildCommit = "-"
	BuildTime   = "-"
)

func VersionString() string {
	return fmt.Sprintf(
		"%v (build-commit=%q build-time=%q)",
		Version, BuildCommit, BuildTime,
	)
}
