package fastget

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"golang.org/x/sync/errgroup"
)

// FastGetter Represents the information required to fastget a file url
type FastGetter struct {
	// Client to be used to make requests
	Client *http.Client
	// URL of the file to be downloaded
	URL string
	// Number of concurrent worked should be used to download file
	Workers int
	// Path of output file to write download
	OutputFile string
	// Headers to be included to while making requests
	Headers map[string]string
	// OnBeforeStart to be called before even download start
	OnBeforeStart func(int64, int64)

	// OnStart to be called on started downloading a chunk / a part
	OnStart func(int, int64)
	// OnProgress to be called on change in progress of downloading a chunk / a part
	OnProgress func(int, int64)
	// OnFinish to be called on finished downloading a chunk / a part
	OnFinish func(int)
}

// chunkInfo holds the information about a chunk to be downloaded
type chunkInfo struct {
	output   io.WriterAt
	url      string
	off, lim int64
	wid      int
	headers  map[string]string
}

// Result represents the result of fastget
type Result struct {
	URL         string
	Size        int64
	OutputFile  *os.File
	ElapsedTime time.Duration
}

// NewFastGetter creates and returns an instance of FastGetter
func NewFastGetter(url string) (*FastGetter, error) {
	fg := &FastGetter{
		Client:     http.DefaultClient,
		URL:        url,
		Workers:    3,
		OutputFile: path.Base(url),
		Headers:    make(map[string]string),
	}
	return fg, nil
}

// Get ultrafast downloads the file
func (fg *FastGetter) Get() (*Result, error) {
	return fg.get(context.Background())
}

// GetCtx ultrafast downloads the file
func (fg *FastGetter) GetCtx(ctx context.Context) (*Result, error) {
	return fg.get(ctx)
}

func (fg *FastGetter) get(ctx context.Context) (*Result, error) {
	canFastGet, length, err := fg.checkEligibility(ctx)
	if err != nil {
		return nil, err
	}
	if !canFastGet {
		log.Println("WARN: FileURL doesn't support parellel download.")
		fg.Workers = 1
	}

	chunkLen := length / int64(fg.Workers)

	if fg.OnBeforeStart != nil {
		fg.OnBeforeStart(length, chunkLen)
	}

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
			return fg.getChunk(ctx, &chunkInfo{
				output: output, url: fg.URL,
				off: off, lim: lim, wid: wid, headers: fg.Headers,
			})
		})

		start = end
	}

	err = wg.Wait()
	if err != nil {
		return nil, err
	}
	elapsed := time.Since(startTime)

	result := &Result{
		URL:         fg.URL,
		Size:        length,
		OutputFile:  output,
		ElapsedTime: elapsed,
	}
	return result, nil
}

func (fg *FastGetter) checkEligibility(ctx context.Context) (bool, int64, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", fg.URL, nil)
	if err != nil {
		return false, 0, err
	}
	for key, value := range fg.Headers {
		req.Header.Add(key, value)
	}

	res, err := fg.Client.Do(req)
	if err != nil {
		return false, 0, err
	}
	defer res.Body.Close()
	acceptRanges := res.Header.Get("Accept-Ranges") == "bytes"
	length := res.ContentLength

	return acceptRanges, length, nil
}

// makeRequest performs the HTTP request and returns the response object
func (cInfo *chunkInfo) makeRequest(ctx context.Context, client *http.Client) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", cInfo.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", cInfo.off, cInfo.lim))
	// Add custom headers
	for key, value := range cInfo.headers {
		req.Header.Add(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// getChunk fetches the part of file to download
func (fg *FastGetter) getChunk(ctx context.Context, cInfo *chunkInfo) error {
	if fg.OnStart != nil {
		fg.OnStart(cInfo.wid, cInfo.lim-cInfo.off)
	}
	resp, err := cInfo.makeRequest(ctx, fg.Client)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("server responded with %d status code, expected %d", resp.StatusCode, http.StatusPartialContent)
	}

	var written int64
	contentLen := resp.ContentLength

	buf := make([]byte, 1*1024*1024)
	for {
		nr, er := resp.Body.Read(buf)

		if nr > 0 {
			nw, err := cInfo.output.WriteAt(buf[0:nr], cInfo.off)
			if err != nil {
				return fmt.Errorf("error writing chunk. %s", err.Error())
			}
			if nr != nw {
				return fmt.Errorf("error writing chunk. written %d, but expected %d", nw, nr)
			}

			cInfo.off = int64(nw) + cInfo.off
			if nw > 0 {
				written += int64(nw)
			}
			if fg.OnProgress != nil {
				fg.OnProgress(cInfo.wid, written)
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
		fg.OnFinish(cInfo.wid)
	}
	return nil
}
