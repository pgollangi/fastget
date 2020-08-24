package cmd

import (
	"fmt"
	"os"
)

// Version is set at build
var version string

// build is set at build
var build string

func main() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
