package dfs

import (
	"bytes"
	"fmt"
)

func CircleDetect(m map[string]string) error {
	g := NewGraph(m)
	g.dfsAll()
	if circles := g.Circles(); len(circles) > 0 {
		return fmt.Errorf("circled on %v", g.Circles())
	}
	return nil
}

// the search path begin from a node, util END or CIRCLED
// 1. searched path
// 2. if the path contains a circle or a subcircle
func PathFromNode(m map[string]string, start string) ([]string, bool) {
	g := NewGraph(m)
	return g.dfs4path(start, nil)
}

// all search path to a node
func AllPathsToNode(m map[string]string, end string) [][]string {
	ret := make([][]string, 0)
	for k := range m {
		path, circled := PathFromNode(m, k)
		if !circled && len(path) > 1 { // ignore circled path & one element path
			if idx := index(path, end); idx > 0 { // the path contains the step we expect
				// ret = append(ret, path[:idx+1])
				ret = append(ret, path[:idx])
				fmt.Println(end, "is @", idx, "of", path, "cut the path:", path[:idx])
			}
		}
	}
	return ret
}

/*
Not Allowed: A -> B, A -> C
Allowed:     A -> B, B -> C, M -> C, N -> C
TODO:        A -> [B, C]
*/
// 模拟一个有向图
// 启动一个节点只能指向一个方向的另外一个节点, 不能同时指向多个节点
// 否则涉及到多个闭环的检测问题
type graph struct {
	// 使用Map来模拟这个有向图, 其中图中的每个节点只能有一个子节点
	// 多个不同节点可以有相同的子节点, 也就是说一个子节点可以有多个不同的父节点
	m map[string]string

	// 如果使用 Map[string][]string 来模拟的话, 就可以实现一个节点包含多个子节点
	c map[string][]string
}

func NewGraph(m map[string]string) *graph {
	initMap := make(map[string]string)
	if m != nil {
		for k, v := range m {
			initMap[k] = v
		}
	}
	return &graph{
		m: initMap,
		c: make(map[string][]string),
	}
}

func (g *graph) Circles() map[string][]string {
	g.dfsAll()
	return g.c
}

func (g *graph) String() string {
	buf := bytes.NewBuffer(nil)
	for k, v := range g.m {
		buf.WriteString(k + " -> " + v + "\n")
	}
	return buf.String()
}

// detect all circles and put into graph.c
func (g *graph) dfsAll() {
	if g == nil {
		return
	}
	for src := range g.m {
		circle := g.dfs4circle(src, src, nil)
		if len(circle) > 0 {
			g.c[src] = circle
		}
	}
}

// 如果有多个闭环怎么办 ?
// 指定一个节点`start`, 开始深度搜索直至最后发生闭环或搜索结束.
// 第一个返回值是发生闭环的路径
// 第二个返回值是走过的所有路径 (不论是否发生闭环)
func (g *graph) dfs4circle(start, current string, stacks []string) []string {
	// check if walked by
	for _, step := range stacks {
		if current == step { // circled, has been walked by
			if current == start { // full circled @ `start`
				return append(stacks, current)
			}
			return nil // sub circled @ any-walked-other-node
		}
	}

	dst, ok := g.m[current]
	if !ok { // search end // 搜索结束
		return nil
	}

	// add a walk step
	walked := append(stacks, current)

	// walk to deeper
	return g.dfs4circle(start, dst, walked)
}

// 指定一个起点`start`, 将整个搜索路径打印出来(非闭环)
// 起点`start`不能是闭环的, 否则将死循环无法退出
func (g *graph) dfs4path(start string, stacks []string) ([]string, bool) {
	// check if walked by
	for _, step := range stacks {
		if start == step { // circled, no matter circle at `start` or any where
			return append(stacks, start), true
		}
	}

	dst, ok := g.m[start]
	if !ok { // search end
		return append(stacks, start), false
	}

	// add a walk step
	walked := append(stacks, start)

	// walk to deeper
	return g.dfs4path(dst, walked)
}

func index(a []string, b string) int {
	for idx, v := range a {
		if v == b {
			return idx
		}
	}
	return -1
}
