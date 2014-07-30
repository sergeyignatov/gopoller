package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path"
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
			out, err := check(t.Url)
			t.Output = out
			fmt.Println(t.Url, t.Error, err)
			if err != nil && !t.Error {
				fmt.Println("new", err)
				t.Error = true
				t.Last = time.Now()
				if len(t.Phone) > 3 {
					fmt.Println("Send SMS to", t.Phone)
					//go sendSMS(t.Phone, out+" : "+t.Url)
				}
				log.Print(t.Url, " ", out)

			} else if err == nil && t.Error {
				log.Print(t.Url, " back to ok")
				t.Error = false
				t.Last = time.Now()
			} else if err != nil {
				t.Error = true
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

	key := "gopoller/tasks"
	fname := path.Join(Settings.Dir, t.Id)
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
	MakeRedisCMD("hset", key, t.Id, b.Bytes())

	if e != nil {
		return e
	}
	return nil

}
func (t *Task) Delete() {
	MakeRedisCMD("hdel", "gopoller/tasks", t.Id)
	os.Remove(path.Join(Settings.Dir, t.Id))
}
