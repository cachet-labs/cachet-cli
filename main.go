package main

import "github.com/cachet-labs/cachet-cli/cmd"

// version is injected by goreleaser: -X main.version={{.Version}}
var version = "dev"

func main() {
	cmd.Execute(version)
}
