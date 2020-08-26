package main

import (
	"fmt"
	"os"

	"github.com/pgollangi/fastget/commands"
)

// Version is set at build
var version string

// build is set at build
var build string

func main() {
	commands.Version = version
	commands.Build = build
	if err := commands.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
