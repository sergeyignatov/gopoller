package models

import (
	"fmt"
	"time"
)

type Task struct {
	ch     chan bool
	url    string
	period uint
}

func NewTask(url string, period uint) *Task {
	return &Task{ch: make(chan bool), url: url, period: period}
}

func (t *Task) Start() {
	for {
		select {
		case <-t.ch:
			fmt.Println("Service stopped")
			return
		default:

		}
		fmt.Println(t.url)
		//do smth
		time.Sleep(time.Duration(t.period) * time.Second)
	}
}
func (t *Task) Stop() {
	close(t.ch)
}
