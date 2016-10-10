package main

import (
	"log"
	"os"
	"time"

	"../../axel"
)

func main() {
	axel, err := axel.New("http://192.168.156.101:81/tmp/10M", "./10M", 10, time.Second*10)
	if err != nil {
		log.Fatalln(err)
	}

	err = axel.Download()
	if err != nil {
		log.Fatalln(err)
	}

	os.Stdout.WriteString("+OK\n")
}
