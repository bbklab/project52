package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"

	"../../emlfile"
)

func main() {
	data, _ := ioutil.ReadFile("sample.eml")
	res, err := emlfile.ParseEml(data)
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
