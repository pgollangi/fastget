package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/pgollangi/fastget"
	"github.com/spf13/cobra"

	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"
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

type barStatus struct {
	iT  time.Time
	bar *mpb.Bar
}

func runCommand(cmd *cobra.Command, args []string) error {
	if ok, _ := cmd.Flags().GetBool("version"); ok {
		executeVersionCmd()
		return nil
	} else if len(args) != 1 {
		cmd.Usage()
		return nil
	}

	threads, _ := cmd.Flags().GetInt("workers")

	url := args[0]

	fg, err := fastget.NewFastGetter(url)

	if err != nil {
		return err
	}
	fg.Workers = threads

	mpbars := make(map[int]*barStatus)

	var counter int

	mp := mpb.New(
		mpb.WithWidth(100),
	)

	fg.OnStart = func(worker int, totalSize int64) {
		bID := counter + 1
		counter = bID
		if bID == 1 {
			fmt.Println("Download started..")
		}

		mpbar := mp.AddBar(totalSize, mpb.BarStyle("[=>-|"),
			mpb.PrependDecorators(
				decor.CountersKiloByte("% 6.2f / % .2f"),
				decor.Percentage(decor.WCSyncSpace),
			),
			mpb.AppendDecorators(
				decor.EwmaETA(decor.ET_STYLE_GO, 90),
				decor.Name(" ] "),
				decor.EwmaSpeed(decor.UnitKB, "% .2f", 60),
			))

		mpbars[worker] = &barStatus{
			bar: mpbar,
			iT:  time.Now(),
		}

	}

	fg.OnProgress = func(worker int, download int64) {
		barStatus := mpbars[worker]

		dur := time.Since(barStatus.iT)

		barStatus.bar.SetCurrent(download)
		barStatus.bar.DecoratorEwmaUpdate(dur)
		barStatus.iT = time.Now()
	}

	result, err := fg.Get()
	if err != nil {
		return err
	}

	mp.Wait()

	pwd, err := os.Getwd()

	oFile := filepath.Join(pwd, result.OutputFile.Name())

	fmt.Printf("Download finished in %s. File: %s", result.ElapsedTime.Round(time.Second), oFile)

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
