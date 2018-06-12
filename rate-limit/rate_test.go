package rate

import (
	"testing"
	"time"

	check "gopkg.in/check.v1"
)

type rateSuit struct{}

var _ = check.Suite(new(rateSuit))

func TestRate(t *testing.T) {
	check.TestingT(t)
}

func (s *rateSuit) TestBasic(c *check.C) {
	l := NewLimiter(time.Second*1, 10) // max 10 events in 1s
	c.Assert(l, check.NotNil)

	// one event
	err := l.Take()
	c.Assert(err, check.IsNil)

	// taken 1, remains 9
	taken, remain := l.Taken(), l.Remains()
	c.Assert(taken, check.Equals, 1)
	c.Assert(remain, check.Equals, 10-1)

	// sleep 1 second
	time.Sleep(time.Second)

	// taken 0, remains 10
	taken, remain = l.Taken(), l.Remains()
	c.Assert(taken, check.Equals, 0)
	c.Assert(remain, check.Equals, 10)
}

func (s *rateSuit) TestOutOfToken(c *check.C) {
	l := NewLimiter(time.Second*1, 10) // max 10 events in 1s
	c.Assert(l, check.NotNil)

	// ten events
	for i := 1; i <= 10; i++ {
		err := l.Take()
		c.Assert(err, check.IsNil)
	}

	// taken 10, remains 0
	taken, remain := l.Taken(), l.Remains()
	c.Assert(taken, check.Equals, 10)
	c.Assert(remain, check.Equals, 10-10)

	// try one event now!
	err := l.Take()
	c.Assert(err, check.NotNil)
	c.Assert(err, check.Equals, ErrNoMoreTokens)

	// sleep 1 second
	time.Sleep(time.Second)

	// taken 0, remains 10
	taken, remain = l.Taken(), l.Remains()
	c.Assert(taken, check.Equals, 0)
	c.Assert(remain, check.Equals, 10)

	// try one event again.
	err = l.Take()
	c.Assert(err, check.IsNil)

	// taken 1, remains 9
	taken, remain = l.Taken(), l.Remains()
	c.Assert(taken, check.Equals, 1)
	c.Assert(remain, check.Equals, 10-1)

}
