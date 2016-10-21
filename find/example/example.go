package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"regexp"

	"../../find"
)

func main() {
	res, err := find.Find("/tmp", find.TypeAll, regexp.MustCompile(`.sock`), false)
	if err != nil {
		log.Fatalln(err)
	}

	pretty(res)
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
