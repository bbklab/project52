 goab
===========
go version of `ab`, a light-weight HTTP bench tool.

Usage
 ------

保持10个并发，持续压测60s，目标URL: http://cn.bing.com
```bash
# ./goab -c 10 -t 60s -u http://cn.bing.com
{
    "200": 2431,
    "duration": 60,
    "rate": 40,
    "succ": 2431,
    "total": 2431
}
```
