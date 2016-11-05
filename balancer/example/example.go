package main

import (
	"fmt"

	"../../balancer"
)

type node struct {
	id     string
	weight int
}

func (n *node) WeightN() int {
	return n.weight
}

func main() {
	nodes := []balancer.Item{
		&node{"node1", 1},
		&node{"node2", 2},
		&node{"node3", 3},
		&node{"node4", 4},
		&node{"node5", 5},
	}

	fmt.Println("--------> RR Balancer")
	rr := balancer.NewRR()
	for i := 1; i <= 10; i++ {
		// try changing the item slice in the halfway
		// if i == 5 {
		// nodes = append(nodes, &node{"node6", 6})
		// }
		next := rr.Next(nodes) // the nodes slice size and order must be fixed
		if next == nil {
			continue
		}
		fmt.Println(i, next.(*node).id)
	}

	fmt.Println("--------> Weight Balancer")
	wr := balancer.NewWeight()
	cnt := make(map[*node]int)
	for i := 1; i <= 10000; i++ {
		next := wr.Next(nodes)
		if next == nil {
			continue
		}
		cnt[next.(*node)]++
	}
	for node, count := range cnt {
		fmt.Println(node.id, node.weight, count)
	}
}
