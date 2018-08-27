package naivecrypto

import (
	"fmt"
	"testing"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(new(labelSuit))

type labelSuit struct{}

func TestLabel(t *testing.T) {
	check.TestingT(t)
}

func (s *labelSuit) TestLabelsBase(c *check.C) {

	var datas = []string{
		"gopkg.in/check.v1",
		"hello world",
	}

	for _, data := range datas {
		encoded := Encode([]byte(data))
		fmt.Println(data, "---->", encoded)
		c.Assert(encoded, check.Not(check.Equals), data)

		decoded := Decode([]byte(encoded))
		fmt.Println(encoded, "---->", decoded)
		c.Assert(decoded, check.Equals, data)
	}
}
