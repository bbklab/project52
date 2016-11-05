package balancer

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// NewRR is exported
func NewRR() Balancer {
	return new(rrBalancer)
}

// NewWeight is exported
func NewWeight() Balancer {
	return new(weightBalancer)
}

//
// interface define and implemention
//

// Balancer is a generic balancer
type Balancer interface {
	Next([]Item) Item
}

// Item is a generic item to be selected
//
// note: the Weight is no use for rrBalancer, only used for weightBalancer
// and any negative weight value will be treated as positive.
// the weight value 0 means the item was disabled, if all of iteams weight equals 0, Next() return nil
type Item interface {
	WeightN() int
}

// note: when using rrBalancer, the items slice Size & Order should be fixed
//
// if item adding / removing occured during multi Next calls,
// the rrBalancer can't ensure each time the returned item is RRed
type rrBalancer struct {
	current int
}

func (b *rrBalancer) Next(items []Item) Item {
	if len(items) == 0 {
		return nil
	}

	if b.current >= len(items) {
		b.current = 0
	}

	t := items[b.current]
	b.current++
	return t
}

type weightBalancer struct{}

func (b *weightBalancer) Next(items []Item) Item {
	if len(items) == 0 {
		return nil
	}

	var wsum = int(0)
	// caculate weight sum first (treat negative -> positive)
	for _, item := range items {
		val := item.WeightN()
		if val < 0 {
			val = -val
		}
		wsum += val
	}

	// if all of weight value equals 0, return nil
	if wsum == 0 {
		return nil
	}

	// get a random between [0-wsum)
	var (
		randval = rand.Intn(wsum)
		n       int
	)
	for _, item := range items {
		n += item.WeightN()
		if n > randval {
			return item
		}
	}
	return nil
}
