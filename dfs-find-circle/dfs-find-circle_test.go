package dfs

import (
	"testing"

	check "gopkg.in/check.v1"
)

type dfsSuit struct{}

var _ = check.Suite(new(dfsSuit))

func TestDFS(t *testing.T) {
	check.TestingT(t)
}

func (s *dfsSuit) TestDFS(c *check.C) {
	testData := []struct {
		m          map[string]string
		hasCircle  bool
		NumCircles int
	}{
		{
			nil, false, 0,
		},
		{
			map[string]string{},
			false,
			0,
		},
		{
			map[string]string{
				"a": "b",
				"c": "d",
				"m": "n",
			},
			false,
			0,
		},
		{
			map[string]string{
				"a": "b",
				"b": "a",
				"m": "n",
			},
			true,
			2,
		},
		{
			map[string]string{
				"a": "a",
				"b": "a",
				"m": "n",
			},
			true,
			1,
		},
		{
			map[string]string{
				"a": "b",
				"b": "c",
				"c": "a",
			},
			true,
			3,
		},
		{
			map[string]string{
				"a": "b",
				"b": "c",
				"c": "d",
				"d": "e",
				"e": "b",
			},
			true,
			4,
		},
	}

	for _, data := range testData {
		hasCircle, circles := fcheck(data.m)
		c.Log(data.m)
		c.Log(circles)
		c.Assert(hasCircle, check.Equals, data.hasCircle)
		c.Assert(len(circles), check.Equals, data.NumCircles)
	}
}

func fcheck(m map[string]string) (bool, map[string][]string) {
	g := NewGraph(m)
	g.dfsAll()
	return len(g.Circles()) > 0, g.Circles()
}
