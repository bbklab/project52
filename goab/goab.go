package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "ab"
	app.Usage = "a simple http stress bench tool like ab"
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:   "concurrency,c",
			Usage:  "number of concurrent requests to keep running",
			Value:  10,
			EnvVar: "CONCURRENCY",
		},
		cli.DurationFlag{
			Name:   "timelimit,t",
			Usage:  "max time to spend on benchmarking",
			Value:  1 * time.Minute,
			EnvVar: "DURATION",
		},
		cli.StringFlag{
			Name:   "url,u",
			Usage:  "target http URL",
			EnvVar: "ADDR",
		},
	}
	app.Action = func(ctx *cli.Context) {
		result, err := doStress(ctx)
		if err != nil {
			logrus.Fatalln(err)
		}
		pretty(result)
	}

	app.RunAndExitOnError()
}

func doStress(ctx *cli.Context) (map[string]int64, error) {
	// verify
	var (
		c = ctx.Int("concurrency")
		d = ctx.Duration("timelimit")
		u = ctx.String("url")
	)
	if c <= 0 {
		cli.ShowSubcommandHelp(ctx)
		return nil, errors.New("bad args: `concurrency`")
	}
	if u == "" {
		cli.ShowSubcommandHelp(ctx)
		return nil, errors.New("bad args: `url`")
	}

	// prepare
	var (
		m      = map[string]int64{} // counter
		mu     sync.Mutex           // protect m
		tokens = make(chan bool, c) // max concurrence
		start  = time.Now()
		end    bool
		idx    int
	)

	time.AfterFunc(d, func() {
		end = true
	})

	defer func() {
		m["rate"] = m["succ"] / int64(d.Seconds())
		m["duration"] = int64(time.Now().Sub(start).Seconds())
	}()

	for {

		if end {
			close(tokens)
			break
		}

		tokens <- true // get token
		idx++

		go func(idx int) {
			defer func() {
				<-tokens // release token
			}()
			resp, err := http.Get(u)
			mu.Lock()
			if err != nil {
				m["error"]++
				logrus.Errorln(idx, err)
			} else {
				m["succ"]++
				m[fmt.Sprintf("%d", resp.StatusCode)]++
				resp.Body.Close()
			}
			m["total"]++
			mu.Unlock()
		}(idx)

	}

	<-tokens
	return m, nil
}

func pretty(data interface{}) error {
	b, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(append(b, '\n'))
	return err
}
