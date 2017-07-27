/*
	mesos event api stressor
*/
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"./mesosproto"

	log "github.com/Sirupsen/logrus"
	"github.com/andygrunwald/megos"
	"github.com/gogo/protobuf/proto"
)

var (
	fid      string // rewrite later
	sid      string // rewrite later
	taskName = "bbklab-stressor"

	addr        string
	total       int
	concurrency int
	image       string
)

func init() {
	if len(os.Args) < 5 {
		log.Fatalln("require args: addr, total, concurrency, image")
	}

	addr = os.Args[1]
	image = os.Args[4]

	var err error

	total, err = strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatalln("invalid total", os.Args[2])
	}

	concurrency, err = strconv.Atoi(os.Args[3])
	if err != nil {
		log.Fatalln("invalid concurrency", os.Args[3])
	}
}

func main() {
	fmt.Println(addr, total, concurrency)

	log.SetFormatter(&log.TextFormatter{})

	client := newClient(addr)

	errCh := make(chan error)
	go func() {
		errCh <- client.subcribe()
	}()

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		for range ch {
			if err := client.teardown(); err != nil {
				fmt.Println("teardown error:", err)
				os.Exit(1)
			}
			fmt.Println("teardown succeed")
			os.Exit(0)
		}
	}()

	if err := client.stress(total); err != nil {
		log.Errorln("stress launch error:", err)
	}
	select {}
}

type client struct {
	*http.Client
	endPoint string
}

func newClient(endpoint string) *client {
	return &client{
		Client: &http.Client{
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout:   time.Second * 10,
					KeepAlive: time.Second * 30,
				}).Dial,
			},
		},
		endPoint: "http://" + endpoint + "/api/v1/scheduler",
	}
}

func (c *client) teardown() error {
	if fid == "" {
		return errors.New("framework not registered")
	}

	call := &mesosproto.Call{
		Type: mesosproto.Call_TEARDOWN.Enum(),
		FrameworkId: &mesosproto.FrameworkID{
			Value: &fid,
		},
	}

	resp, err := c.sendRequest(call)
	if err != nil {
		return err
	}

	if code := resp.StatusCode; code != 202 {
		return fmt.Errorf("send teardown call got %d", code)
	}
	return nil
}

func (c *client) subcribe() error {
	call := &mesosproto.Call{
		Type: mesosproto.Call_SUBSCRIBE.Enum(),
		Subscribe: &mesosproto.Call_Subscribe{
			FrameworkInfo: &mesosproto.FrameworkInfo{
				User:            proto.String("root"),
				Name:            proto.String("bbklab"),
				Principal:       proto.String("bbklab"),
				FailoverTimeout: proto.Float64(60 * 60 * 3),
				Hostname:        proto.String("bbklab-pc"),
				Id: &mesosproto.FrameworkID{
					Value: proto.String(""),
				},
			},
		},
	}

	resp, err := c.sendRequest(call)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var (
		reader = NewReader(resp.Body)
		dec    = json.NewDecoder(reader)
	)
	for {

		var ev *mesosproto.Event
		err := dec.Decode(&ev)
		if err == io.EOF {
			log.Errorln("subscriber EOF")
			break
		}
		if err != nil {
			log.Errorln("stream decode error:", err)
			break
		}

		// subscribed event
		if *ev.Type == mesosproto.Event_SUBSCRIBED {
			fid = *ev.Subscribed.FrameworkId.Value
		}

		if *ev.Type == mesosproto.Event_UPDATE {
			c.ackUpdateEvent(ev)
		}

		pub.Publish(ev)
		// pretty(ev)
	}

	return nil
}

func (c *client) stress(n int) error {
	offer := SubscribeOffer()

	if err := c.launchTasks(n, offer); err != nil {
		return fmt.Errorf("stress().launch error: %v", err)
	}
	time.Sleep(time.Second * 5)

	tasks, err := c.listTasks()
	if err != nil {
		return fmt.Errorf("stress().list error: %v", err)
	}

	if err := c.killTasks(tasks); err != nil {
		return fmt.Errorf("stress().kill error:", err)
	}

	return nil
}

func (c *client) ackUpdateEvent(ev *mesosproto.Event) error {
	call := &mesosproto.Call{
		FrameworkId: &mesosproto.FrameworkID{
			Value: &fid,
		},
		Type: mesosproto.Call_ACKNOWLEDGE.Enum(),
		Acknowledge: &mesosproto.Call_Acknowledge{
			AgentId: ev.Update.Status.GetAgentId(),
			TaskId:  ev.Update.Status.GetTaskId(),
			Uuid:    ev.Update.Status.GetUuid(),
		},
	}

	resp, err := c.sendRequest(call)
	if err != nil {
		return fmt.Errorf("send ack call got err", err)
	}

	if code := resp.StatusCode; code != 202 {
		return fmt.Errorf("send ack call got %d", code)
	}
	return nil
}

func (c *client) listTasks() ([]megos.Task, error) {
	fs, err := FrameworkState()
	if err != nil {
		return nil, err
	}

	return fs.Tasks, nil
}

func (c *client) launchTasks(n int, offer *mesosproto.Offer) error {
	var (
		aid      = *offer.AgentId.Value
		allStart = time.Now()
		tasks    []*mesosproto.TaskInfo
		wg       sync.WaitGroup
		res      struct {
			m map[string][2]string // taskid -> time duration
			sync.Mutex
		}
	)
	res.m = make(map[string][2]string)

	for i := 0; i < n; i++ {
		var (
			id   = fmt.Sprintf("%d-%s", i, taskName)
			task = newTask(id, aid)
		)
		tasks = append(tasks, task) // launching all task in one offer

		// start a task event subscriber ...
		wg.Add(1)
		go func(id string) {
			var (
				start = time.Now()
				err   error
			)

			log.Infof("task %s starting ...", id)

			defer func() {
				res.Lock()
				if err != nil {
					res.m[id] = [2]string{time.Now().Sub(start).String(), err.Error()}
				} else {
					res.m[id] = [2]string{time.Now().Sub(start).String(), ""}
				}
				res.Unlock()
				wg.Done()
			}()

			// subcribe waitting for task's update events here until task finished or met error.
			for {
				ev := SubscribeTaskUpdate(id)
				status := ev.Update.Status
				log.Infof("task %s got update event: %s", id, status.State.String())
				if IsTaskDone(status) {
					log.Infof("task %s final event: %s", id, status.State.String())
					err = DetectError(status) // check if we met an error.
					break
				}
			}

		}(id)
	}

	log.Printf("launching %d tasks", n)

	call := &mesosproto.Call{
		FrameworkId: &mesosproto.FrameworkID{
			Value: &fid,
		},
		Type: mesosproto.Call_ACCEPT.Enum(),
		Accept: &mesosproto.Call_Accept{
			OfferIds: []*mesosproto.OfferID{
				offer.GetId(),
			},
			Operations: []*mesosproto.Offer_Operation{
				&mesosproto.Offer_Operation{
					Type: mesosproto.Offer_Operation_LAUNCH.Enum(),
					Launch: &mesosproto.Offer_Operation_Launch{
						TaskInfos: tasks,
					},
				},
			},
			Filters: &mesosproto.Filters{RefuseSeconds: proto.Float64(1)},
		},
	}

	resp, err := c.sendRequest(call)
	if err != nil {
		return fmt.Errorf("send launch call got err", err)
	}

	if code := resp.StatusCode; code != 202 {
		return fmt.Errorf("send launch call got %d", code)
	}

	wg.Wait()
	pretty(res.m)
	log.Println("stress.launch wait all task finished, cost:", time.Now().Sub(allStart).String())

	return nil
}

func (c *client) killTasks(tasks []megos.Task) error {
	var (
		allStart = time.Now()
		wg       sync.WaitGroup
		res      struct {
			m map[string][2]string // taskid -> time duration
			sync.Mutex
		}
	)
	res.m = make(map[string][2]string)

	for _, task := range tasks {

		wg.Add(1)
		go func(task megos.Task) {
			var (
				start = time.Now()
				err   error
			)

			log.Infof("task %s killing ...", task.ID)

			defer func() {
				res.Lock()
				if err != nil {
					res.m[task.ID] = [2]string{time.Now().Sub(start).String(), err.Error()}
				} else {
					res.m[task.ID] = [2]string{time.Now().Sub(start).String(), ""}
				}
				res.Unlock()
				wg.Done()
			}()

			call := &mesosproto.Call{
				FrameworkId: &mesosproto.FrameworkID{
					Value: &fid,
				},
				Type: mesosproto.Call_KILL.Enum(),
				Kill: &mesosproto.Call_Kill{
					TaskId: &mesosproto.TaskID{
						Value: proto.String(task.ID),
					},
					AgentId: &mesosproto.AgentID{
						Value: proto.String(task.SlaveID),
					},
				},
			}

			resp, err := c.sendRequest(call)
			if err != nil {
				err = fmt.Errorf("send kill call error: %v", err)
				return
			}

			if code := resp.StatusCode; code != 202 {
				err = fmt.Errorf("send kill call on %s got %d", task.Name, code)
				return
			}

			// subcribe waitting for task's update events here until task finished or met error.
			for {
				ev := SubscribeTaskUpdate(task.ID)
				status := ev.Update.Status
				log.Infof("task %s got update event: %s", task.ID, status.State.String())
				if IsTaskDone(status) {
					log.Infof("task %s final event: %s", task.ID, status.State.String())
					err = DetectError(status) // check if we met an error.
					break
				}
			}

		}(task)
	}

	wg.Wait()
	pretty(res.m)
	log.Println("stress.kill wait all task finished, cost:", time.Now().Sub(allStart).String())

	return nil
}

func (c *client) sendRequest(data *mesosproto.Call) (*http.Response, error) {
	bs, err := proto.Marshal(data)
	if err != nil {
		return nil, err
	}

	jbs, _ := json.Marshal(data)
	fmt.Println("sending call ---> ", string(jbs))

	req, err := http.NewRequest("POST", c.endPoint, bytes.NewReader(bs))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Accept", "application/json")
	if sid != "" {
		req.Header.Set("Mesos-Stream-Id", sid)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	if v := resp.Header.Get("Mesos-Stream-Id"); v != "" {
		sid = v
	}

	return resp, nil
}

// utils...
//
//
func pretty(data interface{}) error {
	b, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}
	os.Stdout.Write(append(b, '\n'))
	return nil
}
