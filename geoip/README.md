 GeoIP
===========
lookup asn, country, city for IP address by using GeoIP2 database

Usage
------

```go
    geo, err := geoip.NewGeo("", "")
    if err != nil {
        log.Fatalln(err)
    }

    // query IP geo info
    info := geo.GetGeoInfo("8.8.8.8", "zh-CN")
    pretty(info)

    // update local geo database
    geo.Update()
```
