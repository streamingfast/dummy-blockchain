package main

import (
	"github.com/streamingfast/dummy-blockchain/cmd/dummy-blockchain/app"
)

// Injected at build time
var Version string = "<missing>"

func main() {
	app.Main(Version)
}
