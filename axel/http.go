package axel

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type httpAxel struct {
	// options
	remoteURL string // remote url
	save      string // local save path
	conn      int    // nb of connections, note: maybe reset as 1 if not support partial download

	// http client
	client *http.Client // http(s) client, FIXME prevent potential leaks after dozens of callings ??

	// download runtime fields
	startAt     time.Time  // start time
	totalSize   int64      // total size
	chunkSize   int64      // chunk size
	supportPart bool       // support partial download
	chunksDir   string     // chunks temporarily save dir
	chunkErrs   []string   // held chunk errors  list
	ckerrmux    sync.Mutex // protect above list
	wg          sync.WaitGroup
}

func newHTTPAxel(remoteURL, save string, conn int, connTimeout time.Duration) *httpAxel {
	// set up default parameters
	if conn <= 0 {
		conn = runtime.NumCPU() * 2
	}
	if save == "" {
		save = filepath.Base(remoteURL)
	}

	return &httpAxel{
		remoteURL: remoteURL,
		save:      save,
		conn:      conn,
		client: &http.Client{
			Transport: &http.Transport{
				Dial: func(network, addr string) (net.Conn, error) {
					return net.DialTimeout(network, addr, connTimeout) // with user given connection timeout
				},
				ResponseHeaderTimeout: time.Second * 20,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
		startAt: time.Now(),
	}
}

// Download implement the Axel interface
func (a *httpAxel) Download() error {
	// if save path exists, return conflicts
	if _, err := os.Stat(a.save); err == nil {
		return fmt.Errorf("conflicts, save path [%s] already exists", a.save)
	}

	// get the header & size
	// prepare for the chunk size
	resp, err := a.client.Get(a.remoteURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// return if not http status 200
	if code := resp.StatusCode; code != http.StatusOK {
		return fmt.Errorf("%d - %s", code, http.StatusText(code))
	}

	// detect total size & partial support
	// note: reset conn=1 if total size unknown
	// note: reset conn=1 if not supported partial download
	a.totalSize = resp.ContentLength
	a.supportPart = resp.Header.Get("Accept-Ranges") != ""
	if a.totalSize <= 1 || !a.supportPart {
		a.conn = 1
	}

	// directly download and save
	if a.conn == 1 {
		return a.directSave(resp.Body)
	}

	// partial downloa and save by concurrency
	a.chunkSize = a.totalSize / int64(a.conn)
	a.chunksDir = path.Join(filepath.Dir(a.save), filepath.Base(a.save)+".chunks")
	defer os.RemoveAll(a.chunksDir)

	// concurrency get each chunks
	err = a.getAllChunks()
	if err != nil {
		return err
	}

	// join each pieces to the final save path
	return a.join()
}

func (a *httpAxel) directSave(stream io.ReadCloser) error {
	savefd, err := os.Create(a.save)
	if err != nil {
		return err
	}
	defer savefd.Close()

	_, err = io.Copy(savefd, stream)
	return err
}

func (a *httpAxel) getAllChunks() error {
	a.wg = sync.WaitGroup{}
	a.chunkErrs = make([]string, 0, 0)

	a.wg.Add(a.conn)
	for i := 1; i <= a.conn; i++ {
		go a.getChunk(i)
	}
	a.wg.Wait()

	if len(a.chunkErrs) != 0 {
		return errors.New(strings.Join(a.chunkErrs, ","))
	}
	return nil
}

func (a *httpAxel) getChunk(idx int) {
	defer a.wg.Done()

	// create chunk directory
	err := os.MkdirAll(a.chunksDir, os.FileMode(0755))
	if err != nil {
		a.addChunkErr(newChunkErr(idx, -1, -1, err))
		return
	}

	// construct & send range http request
	var begin = int64((idx - 1)) * a.chunkSize
	var end int64
	if idx == a.conn {
		end = a.totalSize
	} else {
		end = begin + a.chunkSize - 1
	}

	req, err := http.NewRequest("GET", a.remoteURL, nil)
	if err != nil {
		a.addChunkErr(newChunkErr(idx, begin, end, err))
		return
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", begin, end))

	resp, err := a.client.Do(req)
	if err != nil {
		a.addChunkErr(newChunkErr(idx, begin, end, err))
		return
	}
	defer resp.Body.Close()

	// save chunk files
	fchunk := path.Join(a.chunksDir, fmt.Sprintf("%d", idx))
	outfd, err := os.Create(fchunk)
	if err != nil {
		a.addChunkErr(newChunkErr(idx, begin, end, err))
		return
	}
	defer outfd.Close()

	_, err = io.Copy(outfd, resp.Body)
	if err != nil {
		a.addChunkErr(newChunkErr(idx, begin, end, err))
		return
	}
}

func (a *httpAxel) addChunkErr(err error) {
	a.ckerrmux.Lock()
	a.chunkErrs = append(a.chunkErrs, err.Error())
	a.ckerrmux.Unlock()
}

// join all chunk files together as the final save file by order
func (a *httpAxel) join() error {
	savefd, err := os.Create(a.save)
	if err != nil {
		return err
	}
	defer savefd.Close()

	for i := 1; i <= a.conn; i++ {
		fchunk := path.Join(a.chunksDir, fmt.Sprintf("%d", i))
		ckfd, err := os.Open(fchunk)
		if err != nil {
			return err
		}
		defer ckfd.Close()

		_, err = io.Copy(savefd, ckfd)
		if err != nil {
			return err
		}
	}

	return nil
}

type chunkErr struct {
	idx   int
	start int64
	end   int64
	err   error
}

func (e *chunkErr) Error() string {
	return fmt.Sprintf("%d: fetch data chunk %d-%d error: %s", e.idx, e.start, e.end, e.err.Error())
}

func newChunkErr(idx int, start, end int64, err error) *chunkErr {
	return &chunkErr{idx, start, end, err}
}
