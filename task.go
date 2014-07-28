package main

import (
	"fmt"
	"net/http"
	"time"
)

type DayOfWeek struct {
	Name    string
	Enabled bool
}
type Task struct {
	Ch        chan bool
	Id        string
	Url       string
	Period    uint
	Running   bool
	Output    string
	Last      time.Time
	StartTime string
	StopTime  string
	Weekdays  map[int]DayOfWeek
	Phone     string
}

func NewTask(id string, url string, period uint) *Task {
	dow := make(map[int]DayOfWeek)
	dow[0] = DayOfWeek{Name: "Monday"}
	dow[1] = DayOfWeek{Name: "Tuesday"}
	dow[2] = DayOfWeek{Name: "Wednesday"}
	dow[3] = DayOfWeek{Name: "Thusday"}
	dow[4] = DayOfWeek{Name: "Friday"}
	dow[5] = DayOfWeek{Name: "Saturday"}
	dow[6] = DayOfWeek{Name: "Sunday"}
	return &Task{Id: id, Ch: make(chan bool), Url: url, Weekdays: dow, Period: period, StartTime: "8:00", StopTime: "20:00"}
}

func (t *Task) Start() {
	t.Running = true
	t.Ch = make(chan bool)
	transport := http.Transport{
		Dial: dialTimeout,
	}
	client := http.Client{
		Transport: &transport,
	}

	for {
		now := time.Now()

		starttime, err := time.Parse("3:04PM", t.StartTime)
		fmt.Println(starttime, err)
		stoptime, err := time.Parse("3:04PM", t.StopTime)
		fmt.Println(stoptime, err)

		select {
		case <-t.Ch:
			fmt.Println(t.Url, " stopped")
			return
		default:

		}
		if ((60*now.Hour() + now.Minute()) >= (60*starttime.Hour() + starttime.Minute())) && ((60*now.Hour() + now.Minute()) <= (60*stoptime.Hour() + stoptime.Minute())) {
			fmt.Println(t.Url)
			out, err := check(client, t.Url)
			t.Output = out
			t.Last = time.Now()
			if err != nil {
				fmt.Println(err)
			}
			//do smth
		}
		time.Sleep(time.Duration(t.Period) * time.Second)
	}
}
func (t *Task) Stop() {
	t.Running = false
	close(t.Ch)
}
