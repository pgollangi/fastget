package commands

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
)

// Version is the version for netselect
var Version string

// Build holds the date bin was released
var Build string

// RootCmd is the main root/parent command
var RootCmd = &cobra.Command{
	Use:           "fastget $fileURL",
	Short:         "A fastget CLI Tool",
	Long:          `fastget is an open source CLI tool to ultrafast download files over HTTP(s).`,
	SilenceErrors: true,
	SilenceUsage:  true,
	Example: heredoc.Doc(`
		$ fastget https://file.example.com
		$ fastget -v
		`),
	RunE: runCommand,
}

func runCommand(cmd *cobra.Command, args []string) error {
	if ok, _ := cmd.Flags().GetBool("version"); ok {
		executeVersionCmd()
		return nil
	} else if len(args) == 0 {
		cmd.Usage()
		return nil
	}

	return nil
}

// Execute performs fastget command execution
func Execute() error {
	return RootCmd.Execute()
}

func init() {
	RootCmd.Flags().BoolP("version", "v", false, "show fastget version information")
	RootCmd.Flags().BoolP("debug", "d", false, "show debug information")
	RootCmd.Flags().IntP("workers", "t", 1, "use <n> parellel threads")
	RootCmd.Flags().StringP("output", "o", ".", "output file to be written")
}

func executeVersionCmd() {
	fmt.Printf("fast version %s (%s)\n", Version, Build)
	fmt.Println("More info: pgollangi.com/fastget")
}
