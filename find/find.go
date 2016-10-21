package find

import (
	"os"
	"path/filepath"
	"regexp"
)

// nolint
const (
	TypeAll FindType = iota
	TypeDir
	TypeFile
)

// nolint
type FindType int

// Find is like /usr/bin/find
func Find(dir string, typ FindType, reg *regexp.Regexp, one bool) ([]string, error) {
	finder := &finder{dir, typ, reg, one, []string{}}
	return finder.find()
}

type finder struct {
	target string         // condition target directory to be searched
	typ    FindType       // condition type
	reg    *regexp.Regexp // condition regexp to match the name
	one    bool           // condition got one and exit
	res    []string       // result
}

func (f *finder) find() ([]string, error) {
	err := filepath.Walk(f.target, f.walkFunc)
	return f.res, err
}

func (f *finder) walkFunc(path string, info os.FileInfo, err error) error {
	if err != nil {
		return nil // skip error path
	}

	if len(f.res) > 0 && f.one {
		return filepath.SkipDir
	}

	name := info.Name()
	if !f.reg.MatchString(name) {
		return nil // skip non-matched
	}

	switch f.typ {
	case TypeAll:
		f.res = append(f.res, path)
	case TypeDir:
		if info.IsDir() {
			f.res = append(f.res, path)
		}
	case TypeFile:
		if info.Mode().IsRegular() {
			f.res = append(f.res, path)
		}
	}

	return nil
}
