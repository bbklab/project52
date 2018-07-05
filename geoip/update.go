package geoip

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/pkg/archive"

	"../axel"
	"../find"
)

// Update implement Handler interface
func (g *Geo) Update() error {
	// fetch new firstly
	dldir, tarcity, tarasn, err := g.fetch()
	if err != nil {
		return err
	}
	defer os.RemoveAll(dldir)

	// extract the tar files
	filecity, fileasn, err := g.extract(tarcity, tarasn)
	if err != nil {
		return err
	}

	// overwrite update to target path
	err = os.Rename(filecity, g.fcity)
	if err != nil {
		return err
	}

	return os.Rename(fileasn, g.fasn)
}

func (g *Geo) fetch() (basedir, tarcity, tarasn string, err error) {
	basedir, err = ioutil.TempDir(os.TempDir(), "geo-update-dl-")
	if err != nil {
		return
	}

	defer func() {
		if err != nil { // clean up if following steps error
			os.RemoveAll(basedir)
		}
	}()

	tarcity = path.Join(basedir, "city.tgz")
	tarasn = path.Join(basedir, "asn.tgz")

	var (
		urls = [][2]string{
			{cityupdate, tarcity},
			{asnupdate, tarasn},
		}
		errs = make([]string, 0, 0)
		l    sync.Mutex
		wg   sync.WaitGroup
	)

	wg.Add(len(urls))
	for _, pair := range urls {
		go func(pair [2]string) {
			defer wg.Done()

			var (
				url  = pair[0]
				save = pair[1]
				err  error
			)
			defer func() {
				if err != nil {
					l.Lock()
					errs = append(errs, err.Error())
					l.Unlock()
				}
			}()

			axel, err := axel.New(url, save, -1, time.Second*10)
			if err != nil {
				return
			}

			err = axel.Download()
			if err != nil {
				return
			}
		}(pair)
	}
	wg.Wait()

	if len(errs) == 0 {
		err = nil
	} else {
		err = fmt.Errorf("download error: %v", strings.Join(errs, ";"))
	}

	return
}

func (g *Geo) extract(tarcity, tarasn string) (string, string, error) {
	filecity, err := g.extractCity(tarcity)
	if err != nil {
		return "", "", err
	}

	fileasn, err := g.extractAsn(tarasn)
	if err != nil {
		return "", "", err
	}

	return filecity, fileasn, nil
}

func (g *Geo) extractCity(tarcity string) (string, error) {
	return g.extractAndSearch(tarcity, `GeoLite2-City.mmdb$`)
}

func (g *Geo) extractAsn(tarasn string) (string, error) {
	return g.extractAndSearch(tarasn, `GeoLite2-ASN.mmdb$`)
}

func (g *Geo) extractAndSearch(ftar, reg string) (fmmdb string, err error) {
	fd, err := os.Open(ftar)
	if err != nil {
		return
	}
	defer fd.Close()

	dest := filepath.Dir(ftar)
	err = archive.Untar(fd, dest, nil)
	if err != nil {
		return
	}

	res, err := find.Find(dest, find.TypeFile, regexp.MustCompile(reg), true) // find only one file
	if err != nil {
		return
	}

	if len(res) == 0 {
		err = fmt.Errorf("no geo data file matched [%s] found in the tar", reg)
	}

	fmmdb = res[0]
	return
}
