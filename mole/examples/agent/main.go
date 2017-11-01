package main

import (
	"log"
	"time"

	"../../../mole"
)

func main() {
	cfg, err := mole.ConfigFromEnv()
	if err != nil {
		log.Fatalln(err)
	}

	var (
		agent    = mole.NewAgent(cfg)
		delayMin = time.Second      // min retry delay 1s
		delayMax = time.Second * 60 // max retry delay 60s
		delay    = delayMin         // retry delay
	)
	for {
		err := agent.Join()
		if err != nil {
			log.Println("agenr Join() error:", err)
			delay *= 2
			if delay > delayMax {
				delay = delayMax // reset delay to max
			}
			log.Println("agent ReJoin in", delay.String())
			time.Sleep(delay)
			continue
		}

		log.Println("agent Joined succeed, ready ...")
		delay = delayMin // reset dealy to min
		err = agent.Serve()
		if err != nil {
			log.Println("agent Serve() error:", err)
		}
	}
}
