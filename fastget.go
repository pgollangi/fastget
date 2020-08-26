package fastget

import (
	"context"
	"fmt"
	"io"
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
func (fg FastGetter) Get() (*Result, error) {
	return fg.get()
}

func (fg FastGetter) get() (*Result, error) {
	canFastGet, length, err := fg.validateFastGet()
	if err != nil {
		return nil, err
	}
	if !canFastGet {
		// warn
		fmt.Println("WARN: FileURL doesn't support parellel download.")
		fg.Workers = 1
	}

	chunkLen := length / int64(fg.Workers)

	ctx := context.Background()
	client := http.DefaultClient

	output, err := os.OpenFile(fg.OutputFile, os.O_CREATE|os.O_RDWR, 0666)

	if err != nil {
		return nil, err
	}

	wg, ctx := errgroup.WithContext(ctx)

	start := time.Now()

	for off := int64(0); off < length; off += chunkLen {
		off := off
		lim := off + chunkLen
		if lim >= length {
			lim = length
		}
		wg.Go(func() error {
			return getChunk(ctx, client, output, fg.FileURL, off, lim)
		})
	}
	wg.Wait()
	elapsed := time.Since(start)

	r := &Result{
		FileURL:     fg.FileURL,
		Size:        length,
		OutputFile:  output,
		ElapsedTime: elapsed,
	}
	return r, nil
}

func (fg FastGetter) validateFastGet() (bool, int64, error) {
	res, err := http.Head(fg.FileURL)
	if err != nil {
		return false, 0, err
	}
	acceptRanges := res.Header.Get("Accept-Ranges") == "bytes"
	length := res.ContentLength

	return acceptRanges, length, nil
}

func getChunk(ctx context.Context, client *http.Client, file *os.File, url string, offset, limit int64) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, limit))
	resp, err := ctxhttp.Do(ctx, client, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("server responded with %d status code, expected %d", resp.StatusCode, http.StatusPartialContent)
	}
	var written int64
	contentLen := resp.ContentLength

	buf := make([]byte, 4*1024)

	for {
		nr, er := resp.Body.Read(buf)

		if nr > 0 {
			nw, err := file.WriteAt(buf[0:nr], offset)
			if err != nil {
				return fmt.Errorf("error writing chunk. %s", err.Error())
			}
			if nr != nw {
				return fmt.Errorf("error writing chunk. written %d, but expected %d", nw, nr)
			}

			offset = int64(nw) + offset
			if nw > 0 {
				written += int64(nw)
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
	return nil
}

type sectionWriter struct {
	w   io.WriterAt
	off int64
}

func (w *sectionWriter) Write(p []byte) (n int, err error) {
	n, err = w.w.WriteAt(p, w.off)
	w.off += int64(n)
	return
}
