package rate

import (
	"fmt"
	"sync"
	"time"
)

// NewLimiter new a event limiter
func NewLimiter(w time.Duration, n int) Limiter {
	l := &limiter{}
	l.SetLimit(w, n)
	l.tokens = make([]time.Time, 0, 0)
	return l
}

type limiter struct {
	sync.RWMutex               // protect followings
	window       time.Duration // token time window
	limit        int           // token limit
	tokens       []time.Time   // tokens already been taken
}

// Take implement Limiter interface
func (l *limiter) Take() error {
	l.Lock()
	defer l.Unlock()

	l.gc()
	if l.remains() == 0 {
		return ErrNoMoreTokens
	}

	l.push()
	return nil
}

// Remains implement Limiter interface
func (l *limiter) Remains() int {
	l.RLock()
	defer l.RUnlock()

	l.gc()
	return l.remains()
}

// Taken implement Limiter interface
func (l *limiter) Taken() int {
	l.RLock()
	defer l.RUnlock()

	l.gc()
	return l.size()
}

// SetLimit implement Limiter interface
func (l *limiter) SetLimit(w time.Duration, n int) {
	l.Lock()
	if w < 0 {
		w = -w
	}
	if n < 0 {
		n = -n
	}
	l.window = w
	l.limit = n
	l.Unlock()
}

// String implement Limiter interface
func (l *limiter) String() string {
	return fmt.Sprintf("limit %d tokens in %s, current remains %d", l.limit, l.window.String(), l.Remains())
}

//
// unsafe ops
//
func (l *limiter) size() int {
	return len(l.tokens)
}

func (l *limiter) remains() int {
	return l.limit - l.size()
}

// gc shift all of outdated tokens
// the returned value means the nb of tokens been GCed
func (l *limiter) gc() int {
	if len(l.tokens) == 0 {
		return 0
	}

	// the outdated time boundary
	outdated := time.Now().Add(-l.window)

	// range from the head oldest tokens to find out the first non-outdated tokens
	// the index of it should be the nb of tokens to be GCed
	var n int
	for idx, token := range l.tokens {
		if !token.Before(outdated) {
			break
		} else {
			n = idx + 1
		}
	}

	if n > 0 {
		l.shiftN(n)
	}

	return n
}

func (l *limiter) pushN(n int) {
	for i := 1; i <= n; i++ {
		l.push()
	}
}

func (l *limiter) push() {
	l.tokens = append(l.tokens, time.Now())
}

func (l *limiter) shiftN(n int) {
	if l.size() >= n {
		l.tokens = l.tokens[n:]
	}
}

func (l *limiter) shift() {
	if l.size() >= 1 {
		l.tokens = l.tokens[1:]
	}
}
