package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"../../ffmpeg"
)

func main() {

	files := []string{
		"/tmp/maptrack-playback.mp4",
	}

	ffh := ffmpeg.NewHandler()

	for _, file := range files {
		fmt.Println("=======>", file)

		// probe
		info, err := ffh.Probe(file)
		if err != nil {
			log.Fatalln(err)
		}
		pretty(info)

		// extract as gif
		fgif, err := ffh.ExtractGif(file)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println("+OK: EXTRACT GIF:", fgif)

		// convert
		in := file
		out := "m3u8/index.m3u8"
		progch, donech, err := ffh.ToHLS(in, out, false)
		if err != nil {
			log.Fatalln(err)
		}
		for prog := range progch {
			fmt.Println(fmt.Sprintf("%0.2f", prog.Progress)+"%", prog.FramesProcessed)
		}
		fmt.Println("+OK: CONVERT DONE", <-donech)

		// convert with key encrypt
		in = file
		out = "m3u8_enc/index.m3u8"
		progch, donech, err = ffh.ToHLS(in, out, true)
		if err != nil {
			log.Fatalln(err)
		}
		for prog := range progch {
			fmt.Println(fmt.Sprintf("%0.2f", prog.Progress)+"%", prog.FramesProcessed)
		}
		fmt.Println("+OK: CONVERT WITH KEY DONE", <-donech)
	}

	select {}
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
