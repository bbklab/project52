 Naive Crypto
===========
a small naive private crypto tool

Usage
------

```go
    data := "hello world"
    encoded := Encode([]byte(data))     // cW983vI8:7Uo7N3gbHWtcH9he3:zcHR><:654
    decoded := Decode([]byte(encoded))  // hello world
```
