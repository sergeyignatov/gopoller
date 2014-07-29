package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/fzzy/radix/redis"
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
	Error     bool
	Weekdays  map[int]DayOfWeek
	Phone     string
}

func NewTask(id string, url string, period uint) *Task {
	dow := make(map[int]DayOfWeek)
	dow[1] = DayOfWeek{Name: "Monday"}
	dow[2] = DayOfWeek{Name: "Tuesday"}
	dow[3] = DayOfWeek{Name: "Wednesday"}
	dow[4] = DayOfWeek{Name: "Thursday"}
	dow[5] = DayOfWeek{Name: "Friday"}
	dow[6] = DayOfWeek{Name: "Saturday"}
	dow[7] = DayOfWeek{Name: "Sunday"}
	return &Task{Id: id, Ch: make(chan bool), Url: url, Weekdays: dow, Period: period, StartTime: "8:00AM", StopTime: "8:00PM"}
}

func (t *Task) Start() {
	t.Running = true
	log.Print(t.Url, " started")
	t.Ch = make(chan bool)
	transport := http.Transport{
		Dial: dialTimeout,
	}
	client := http.Client{
		Transport: &transport,
	}
	intime := false
	for {
		now := time.Now()
		intime = false
		starttime, err := time.Parse("3:04PM", t.StartTime)
		if err != nil {
			fmt.Println(starttime, err)
		}
		stoptime, err := time.Parse("3:04PM", t.StopTime)
		if err != nil {
			fmt.Println(stoptime, err)
		}

		select {
		case <-t.Ch:
			log.Print(t.Url, " stopped")
			return
		default:

		}
		for _, val := range t.Weekdays {
			if val.Enabled && val.Name == now.Weekday().String() {
				intime = true
				break
			}
		}
		if intime && ((60*now.Hour() + now.Minute()) >= (60*starttime.Hour() + starttime.Minute())) && ((60*now.Hour() + now.Minute()) <= (60*stoptime.Hour() + stoptime.Minute())) {
			out, err := check(client, t.Url)
			t.Output = out
			t.Last = time.Now()
			if err != nil && !t.Error {
				fmt.Println(err)
				t.Error = true
				if len(t.Phone) > 3 {
					go sendSMS(t.Phone, out+" : "+t.Url)
				}
				log.Print(t.Url, ' ', out)

			} else if err == nil && t.Error {
				log.Print(t.Url, " back to ok")
				t.Error = false
			} else {
				t.Error = false
			}
			//do smth
			t.Save()
		}
		time.Sleep(time.Duration(t.Period) * time.Second)
	}
}
func (t *Task) Stop() {
	t.Running = false
	close(t.Ch)
}

func (t *Task) Save() (err error) {
	client, err := redis.DialTimeout("tcp", "127.0.0.1:6379", time.Duration(10)*time.Second)
	defer client.Close()
	if err != nil {
		log.Println("failed to create the client", err)
		return err
	}
	client.Cmd("select", Settings.RedisDB)

	key := "gopoller/tasks"
	fname := "/tmp/states/" + t.Id
	b := new(bytes.Buffer)
	enc := gob.NewEncoder(b)
	err = enc.Encode(t)
	if err != nil {
		return err
	}

	fh, eopen := os.OpenFile(fname, os.O_CREATE|os.O_WRONLY, 0666)
	defer fh.Close()
	if eopen != nil {
		return eopen
	}
	_, e := fh.Write(b.Bytes())
	client.Cmd("hset", key, t.Id, b.Bytes())

	if e != nil {
		return e
	}
	return nil

}
func (t *Task) Delete() {
	client, err := redis.DialTimeout("tcp", "127.0.0.1:6379", time.Duration(10)*time.Second)
	defer client.Close()
	if err != nil {
		log.Println("failed to create the client", err)
		return
	}
	client.Cmd("select", Settings.RedisDB)

	client.Cmd("hdel", "gopoller/tasks", t.Id)
	//os.Remove("/tmp/states/" + t.Id)
}
