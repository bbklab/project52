package main

import (
	"fmt"

	dfs "../../dfs-find-circle"
)

func main() {
	m := map[string]string{
		"a": "b",
		"b": "c",
		"c": "d",
		"d": "m",
		"m": "c",
		"k": "n",
		"l": "n",
		"x": "c",
		"o": "c",
		"s": "r",
		"r": "n",
	}
	g := dfs.NewGraph(m)
	fmt.Println(g)

	fmt.Println("Circles() ---------")
	fmt.Println(g.Circles())

	fmt.Println()
	fmt.Println("PathFromNode() ---------")
	fmt.Println(dfs.PathFromNode(m, "a"))
	fmt.Println(dfs.PathFromNode(m, "k"))
	fmt.Println(dfs.PathFromNode(m, "m"))
	fmt.Println(dfs.PathFromNode(m, "d"))
	fmt.Println(dfs.PathFromNode(m, "x"))

	fmt.Println()
	fmt.Println("AllPathsToNode() ---------")
	fmt.Println(dfs.AllPathsToNode(m, "n"))
}
