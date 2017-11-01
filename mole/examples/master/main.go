package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"../../../mole"
)

func main() {
	cfg, err := mole.ConfigFromEnv()
	if err != nil {
		log.Fatalln(err)
	}

	master := mole.NewMaster(cfg)

	errCh := make(chan error)
	go func() {
		errCh <- master.Serve()
	}()

	go func() {
		for ; ; time.Sleep(time.Second * 5) {
			for id, agent := range master.Agents() {
				requestNode(id, agent)
			}
		}
	}()

	log.Fatalln("master Serve() error:", <-errCh)
}

func requestNode(id string, agent *mole.ClusterAgent) {
	log.Println("requesting on node", id)

	client := agent.Client()

	// request on agent http svr
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s/hello", id), nil)
	req.Close = true
	req.Header.Set("Connection", "close")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("request node /info: error", err)
		return
	}
	defer resp.Body.Close()

	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("request node /info: error", err)
		return
	}
	log.Println("request node /info got response", string(bs))

	// request on agent backend service (docker remote api)
	req, _ = http.NewRequest("GET", fmt.Sprintf("http://%s/version", id), nil)
	req.Close = true
	req.Header.Set("Connection", "close")

	resp, err = client.Do(req)
	if err != nil {
		log.Println("request node backend docker api /version: error", err)
		return
	}
	defer resp.Body.Close()

	bs, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("request node backend docker api /version: error", err)
		return
	}
	log.Println("request node backend docker api /version got response", string(bs))
}
