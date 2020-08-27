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

//ProgressUpdate respresnets
type ProgressUpdate struct {
	TotalSize  int64
	Downloaded int64
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
	canFastGet, length, err := fg.validateFastGet()
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
		limit := end

		wg.Go(func() error {
			fg.OnStart(wid, limit-off)
			return getChunk(ctx, client, output, fg.FileURL, off, limit, fg, wid)
		})

		start = end
	}

	err = wg.Wait()
	if err != nil {
		return nil, err
	}
	elapsed := time.Since(startTime)

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

func getChunk(ctx context.Context, client *http.Client, file *os.File, url string, offset, limit int64,
	fg *FastGetter, wid int) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	// fmt.Println("Getting ", offset, limit)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, limit))
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
	// fmt.Println(limit - offset)
	// buf := make([]byte, limit-offset)

	// _, err = io.Copy(&chunkWriter{
	// 	file: file,
	// 	off:  offset}, resp.Body)
	// fmt.Println("WRITTEN ", offset, limit, wn)
	// return err

	for {
		nr, er := resp.Body.Read(buf)

		// fmt.Println("READ ", nr)

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
			fg.OnProgress(wid, written)
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
	fg.OnFinish(wid)
	return nil
}

type chunkWriter struct {
	file *os.File
	off  int64
}

func (cw *chunkWriter) Write(p []byte) (n int, err error) {
	n, err = cw.file.WriteAt(p, cw.off)
	cw.off += int64(n)
	return
}
