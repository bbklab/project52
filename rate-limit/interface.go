package rate

import (
	"errors"
	"time"
)

// nolint
var (
	ErrNoMoreTokens = errors.New("no more tokens, pls wait and retry")
)

// Limiter controls how frequently events are allowed to happen
type Limiter interface {
	Take() error // Take take one token, if met error, must be ErrNoMoreTokens

	Remains() int // how many tokens remains, always used to check if reached limit line

	Taken() int // how many tokens has been taken

	SetLimit(w time.Duration, n int) // change limit settings on fly

	String() string // print limiter text message
}
