package main

import (
	"encoding/json"
	"fmt"
	"os"

	"../../ordermap"
)

func main() {
	m := ordermap.New()
	m.Set("n", 100)
	m.Set("f", 99.99)
	m.Set("d", "barfoo")
	m.Set("c", "foobar")
	m.Set("b", "bar")
	m.Set("a", "foo")

	for _, key := range m.Keys() {
		fmt.Println(key, m.Get(key))
	}
	json.NewEncoder(os.Stdout).Encode(m)

	m.Del("c")
	for _, key := range m.Keys() {
		fmt.Println(key, m.Get(key))
	}
	json.NewEncoder(os.Stdout).Encode(m)
}
