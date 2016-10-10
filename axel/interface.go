package axel

import (
	"errors"
	"strings"
	"time"
)

var (
	// ErrNotImplemented is exported
	ErrNotImplemented = errors.New("not implemented yet")
	// ErrNotSupported is exported
	ErrNotSupported = errors.New("url type not supported")
)

// Axel is the go port of axel, a light download accelerator
type Axel interface {
	Download() error
}

// New initialize an Axel according by given URL with known protocol
func New(remoteURL, save string, conn int, connTimeout time.Duration) (Axel, error) {
	switch {
	case strings.HasPrefix(remoteURL, "http://") || strings.HasPrefix(remoteURL, "https://"): // support http & https
		return newHTTPAxel(remoteURL, save, conn, connTimeout), nil

	case strings.HasPrefix(remoteURL, "ftp://"):
		return nil, ErrNotImplemented

	default:
		return nil, ErrNotSupported
	}
}
