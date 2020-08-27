package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/MakeNowJust/heredoc"
	"github.com/pgollangi/fastget"
	"github.com/spf13/cobra"

	"github.com/cheggaaa/pb"
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
	} else if len(args) != 1 {
		cmd.Usage()
		return nil
	}

	threads, _ := cmd.Flags().GetInt("threads")

	url := args[0]

	fg, err := fastget.NewFastGetter(url)

	if err != nil {
		return err
	}
	fg.Workers = threads

	bars := make(map[int]*pb.ProgressBar)

	pbPool := pb.NewPool()

	var counter int

	fg.OnStart = func(worker int, totalSize int64) {
		bID := counter + 1
		counter = bID
		if bID == 1 {
			fmt.Println("Download started..")
		}
		bar := pb.New64(totalSize).Prefix(fmt.Sprintf("Part %d ", bID))
		bar.ShowSpeed = true
		bar.ShowElapsedTime = true
		bar.ShowPercent = true
		bar.SetMaxWidth(100)
		bar.SetUnits(pb.U_BYTES_DEC)
		bars[worker] = bar
		pbPool.Add(bar)
		if counter == threads {
			pbPool.Start()
		}
	}

	fg.OnProgress = func(worker int, download int64) {
		bars[worker].Set64(download)

	}

	fg.OnFinish = func(worker int) {
		bars[worker].Finish()
	}

	result, err := fg.Get()
	if err != nil {
		return err
	}

	pbPool.Stop()

	pwd, err := os.Getwd()

	oFile := filepath.Join(pwd, result.OutputFile.Name())

	fmt.Printf("Download finished in %s. File: %s", result.ElapsedTime, oFile)

	return nil
}

// Execute performs fastget command execution
func Execute() error {
	return RootCmd.Execute()
}

func init() {
	RootCmd.Flags().BoolP("version", "v", false, "show fastget version information")
	RootCmd.Flags().BoolP("debug", "d", false, "show debug information")
	RootCmd.Flags().IntP("workers", "w", 3, "use <n> parellel threads")
	RootCmd.Flags().StringP("output", "o", ".", "output file to be written")
}

func executeVersionCmd() {
	fmt.Printf("fast version %s (%s)\n", Version, Build)
	fmt.Println("More info: pgollangi.com/fastget")
}
