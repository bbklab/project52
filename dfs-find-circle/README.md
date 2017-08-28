 DFS Find Circle
===========
use `DFS` (depth first search) algorithm to detect graph circle.

Scenario
------
  * detect dependency circle within Docker Compose services

Usage
------
```go
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
    fmt.Println(g.Circles())  // [c:[c d m c] d:[d m c d] m:[m c d m]]
```

Reference
------
  * https://en.wikipedia.org/wiki/Depth-first_search
