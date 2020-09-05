package fastget

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"time"

	"golang.org/x/net/context/ctxhttp"
	"golang.org/x/sync/errgroup"
)

// FastGetter Represents the information required to fastget a file url
type FastGetter struct {
	FileURL    string
	Workers    int
	OutputFile string
	OnStart    func(int, int64)
	OnProgress func(int, int64)
	OnFinish   func(int)
}

// Result represents the result of fastget
type Result struct {
	FileURL     string
	Size        int64
	OutputFile  *os.File
	ElapsedTime time.Duration
}

// NewFastGetter creates and returns an instance of FastGetter
func NewFastGetter(fileURL string) (*FastGetter, error) {
	fg := &FastGetter{
		FileURL:    fileURL,
		Workers:    3,
		OutputFile: path.Base(fileURL),
	}
	return fg, nil
}

// Get ultrafast downloads the file
func (fg *FastGetter) Get() (*Result, error) {
	return fg.get()
}

func (fg *FastGetter) get() (*Result, error) {
	canFastGet, length, err := fg.checkEligibility()
	if err != nil {
		return nil, err
	}
	if !canFastGet {
		// warn
		fmt.Println("WARN: FileURL doesn't support parellel download.")
		fg.Workers = 1
	}

	chunkLen := int64(length / int64(fg.Workers))

	ctx := context.Background()
	client := http.DefaultClient

	output, err := os.OpenFile(fg.OutputFile, os.O_CREATE|os.O_RDWR, 0666)

	if err != nil {
		return nil, err
	}

	wg, ctx := errgroup.WithContext(ctx)

	startTime := time.Now()

	var start, end int64
	for wid := 1; wid <= fg.Workers; wid++ {

		if wid == fg.Workers {
			end = length // last part
		} else {
			end = start + chunkLen
		}

		wid := wid
		off := start
		lim := end

		wg.Go(func() error {
			return fg.getChunk(ctx, client, output, fg.FileURL, off, lim, wid)
		})

		start = end
	}

	err = wg.Wait()
	if err != nil {
		return nil, err
	}
	elapsed := time.Since(startTime)

	result := &Result{
		FileURL:     fg.FileURL,
		Size:        length,
		OutputFile:  output,
		ElapsedTime: elapsed,
	}
	return result, nil
}

func (fg FastGetter) checkEligibility() (bool, int64, error) {
	res, err := http.Head(fg.FileURL)
	if err != nil {
		return false, 0, err
	}
	acceptRanges := res.Header.Get("Accept-Ranges") == "bytes"
	length := res.ContentLength

	return acceptRanges, length, nil
}

func (fg FastGetter) getChunk(
	ctx context.Context, client *http.Client, file *os.File, url string, off, lim int64, wid int) error {
	if fg.OnStart != nil {
		fg.OnStart(wid, lim-off)
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	// fmt.Println("Getting ", offset, limit)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", off, lim))
	resp, err := ctxhttp.Do(ctx, client, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("server responded with %d status code, expected %d", resp.StatusCode, http.StatusPartialContent)
	}
	// fmt.Println("GOT ", offset, limit)
	var written int64
	contentLen := resp.ContentLength

	buf := make([]byte, 1*1024*1024)
	for {
		nr, er := resp.Body.Read(buf)

		if nr > 0 {
			nw, err := file.WriteAt(buf[0:nr], off)
			if err != nil {
				return fmt.Errorf("error writing chunk. %s", err.Error())
			}
			if nr != nw {
				return fmt.Errorf("error writing chunk. written %d, but expected %d", nw, nr)
			}

			off = int64(nw) + off
			if nw > 0 {
				written += int64(nw)
			}
			if fg.OnProgress != nil {
				fg.OnProgress(wid, written)
			}
		}

		if er != nil {
			if er.Error() == "EOF" {
				if contentLen == written {
					// Download successfully
				} else {
					return fmt.Errorf("error reading response. %s", er.Error())
				}
				break
			}
			return er
		}

	}
	if fg.OnFinish != nil {
		fg.OnFinish(wid)
	}
	return nil
}
