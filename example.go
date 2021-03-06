package main

import (
	"net/http"
	"encoding/json"
	_ "net/http/pprof"
	"delaytask"
	"time"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"utils/trace"
)

type OncePingTask struct {
	delaytask.Task
	Url string `json:"Url"`
}

func (t *OncePingTask) Run() (bool, error) {
	resp, err := http.Get(t.Url)
	if err != nil {
		return false, err
	}
	t.RunAt = delaytask.TaskTime(time.Now())
	delaytask.Logger.WithFields(logrus.Fields{
		"id":      t.GetID(),
		"RunAt":   t.GetRunAt(),
		"ToRunAt": t.GetToRunAt(),
	}).Infoln("OncePingTask ToRunAt RunAt")

	defer resp.Body.Close()
	return true, nil
}

func (t *OncePingTask) ToJson() string {
	b, err := json.Marshal(t)
	if err != nil {
		return ""
	}
	return string(b)
}

type PeriodPingTask struct {
	delaytask.PeriodicTask
	Url string `json:"Url"`
}

func (t *PeriodPingTask) Run() (bool, error) {
	resp, err := http.Get(t.Url)
	defer resp.Body.Close()
	if err != nil {
		return false, err
	}
	ioutil.ReadAll(resp.Body)
	delaytask.Logger.WithFields(logrus.Fields{
		"id":  t.GetID(),
		"err": err,
	}).Infoln("PeriodPingTask Run")
	return true, nil
}
func (t *PeriodPingTask) ToJson() string {
	b, err := json.Marshal(t)
	if err != nil {
		return ""
	}
	return string(b)
}

func main() {
	engine := delaytask.NewEngine("1s", 10, "redis://:uestc12345@127.0.0.1:6379/4",
		"messageQ", "remote-task0:")
	engine.AddTaskCreator("OncePingTask", func(task string) delaytask.Runner {
		p := &OncePingTask{}
		if err := json.Unmarshal([]byte(task), p); err != nil {
		} else {
			return p
		}
		return nil
	})
	engine.AddTaskCreator("PeriodPingTask", func(task string) delaytask.Runner {
		t := &PeriodPingTask{}
		if err := json.Unmarshal([]byte(task), t); err != nil {
			return nil
		} else {
			return t
		}
	})
	//tracer := trace.NewTrace(0x222)
	//runInterval := time.Second * 50
	//toRunAt := time.Now().Add(time.Minute * 2)
	//t := &PeriodPingTask{
	//	PeriodicTask: delaytask.PeriodicTask{
	//		Task: delaytask.Task{
	//			ID:      tracer.GetID().Int64(),
	//			Name:    "PeriodPingTask",
	//			ToRunAt: delaytask.TaskTime(toRunAt),
	//			Done:    0,
	//			Timeout: delaytask.TaskDuration(time.Second * 5),
	//		},
	//		Interval: delaytask.TaskDuration(runInterval),
	//		EndTime:  delaytask.TaskTime(time.Now().Add(time.Hour * 24 * 365)),
	//	},
	//	Url: "http://www.baidu.com",
	//}
	engine.Start()

	//go func() {
	//	http.ListenAndServe("0.0.0.0:6060", nil)
	//}()

	sigChan := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGTSTP, syscall.SIGHUP)
	go func() {
		sig := <-sigChan
		switch sig {
		case syscall.SIGINT:
			fallthrough
		case syscall.SIGTERM:
			fallthrough
		case syscall.SIGKILL:
			fallthrough
		case syscall.SIGTSTP:
			fallthrough
		case syscall.SIGHUP:
			delaytask.Logger.WithFields(logrus.Fields{
			}).Warnln("engine will stop!!")
			engine.Stop()
		default:

		}
		done <- true
		close(done)
	}()
	<-done
	close(sigChan)
}
