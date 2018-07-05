package main

import (
	"encoding/json"
	"io"
	"log"
	"os"

	"../../geoip"
)

func main() {
	geo, err := geoip.NewGeo("", "")
	if err != nil {
		log.Fatalln(err)
	}

	// query IP geo info
	info := geo.GetGeoInfo("8.8.8.8", "zh-CN")
	pretty(info)

	// update local geo database
	geo.Update()
}

func pretty(data interface{}) error {
	w := io.Writer(os.Stdout)

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "    ")
	err := enc.Encode(data)
	if err != nil {
		return err
	}

	w.Write([]byte{'\r', '\n'})
	return nil
}
