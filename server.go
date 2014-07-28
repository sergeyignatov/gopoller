package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
)

var timeout = time.Duration(2 * time.Second)

func dialTimeout(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, timeout)
}

//var TaskMap = make(map[string]models.Task)
type statemap map[string]*Task

var TaskMap = make(statemap)

func check(client http.Client, url string) (string, error) {
	resp, err := client.Get(url)
	var out string
	if err != nil {
		out = "host is unreachable"
	} else {

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			if string(body) != "ok" {
				out = "wrong answer"
				err = errors.New(out)
			} else {
				out = string(body)
			}
		}
	}
	return out, err
}
func (t statemap) StoreState() (err error) {
	fname := "/tmp/state.dmp"
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
	if e != nil {
		return e
	}
	return nil
}

func LoadState(fname string) (statemap, error) {
	fh, err := os.Open(fname)
	t := make(statemap)
	if err != nil {
		return t, err
	}
	dec := gob.NewDecoder(fh)
	err = dec.Decode(&t)
	defer fh.Close()
	if err != nil {
		return nil, err
	}
	return t, nil
}

func indexHandler(rnd render.Render) {

	tasks := []Task{}
	/*for _, doc := range postDocuments {
		post := models.Post{doc.Id, doc.Title, doc.ContentHtml, doc.ContentMarkdown}
		posts = append(posts, post)
	}*/
	for _, task := range TaskMap {
		tasks = append(tasks, *task)
	}

	rnd.HTML(200, "index", tasks)
}
func toggleHandler(rnd render.Render, r *http.Request, params martini.Params) {
	id := params["id"]
	task := TaskMap[id]
	if task.Running {
		//task.Running = false
		task.Stop()
	} else {
		//task.Running = true
		//task.Ch = make(chan bool)
		go task.Start()
	}
	TaskMap[id] = task
	TaskMap.StoreState()
	rnd.Redirect("/")
}

func editHandler(rnd render.Render, r *http.Request, params martini.Params) {
	task := TaskMap[params["id"]]
	rnd.HTML(200, "add", task)
}
func deleteHandler(rnd render.Render, r *http.Request, params martini.Params) {
	id := params["id"]
	task, ok := TaskMap[id]
	if ok {
		if task.Running {
			task.Stop()
		}
		delete(TaskMap, id)
	}
	TaskMap.StoreState()
	rnd.Redirect("/")
}

func storeHandler(rnd render.Render, r *http.Request, params martini.Params) {
	id := r.FormValue("id")
	url := r.FormValue("url")
	starttime := r.FormValue("starttime")
	stoptime := r.FormValue("stoptime")
	weekdays := r.Form["weekdays"]
	fmt.Println(weekdays)
	p, err := strconv.Atoi(r.FormValue("period"))
	var period uint = 30
	if err == nil {
		period = uint(p)
	}
	if id != "" {
		task := TaskMap[id]
		if task.Running {
			task.Stop()
		}
		task.Url = url
		task.Period = period
		task.StartTime = starttime
		task.StopTime = stoptime
		for _, val := range weekdays {
			ival, _ := strconv.Atoi(val)
			tmp := task.Weekdays[ival]
			tmp.Enabled = true
			task.Weekdays[ival] = tmp
		}
		//task.Weekdays = weekdays
		TaskMap[id] = task
	} else {
		id = GenerateId()
		task := NewTask(id, url, period)
		//task.Weekdays = weekdays
		TaskMap[id] = task
	}
	TaskMap.StoreState()

	rnd.Redirect("/")
}

func addHandler(rnd render.Render) {
	id := GenerateId()
	task := NewTask(id, "", 30)
	TaskMap[id] = task
	//task := Task{Period: 30, StartTime: "8:00", StopTime: "20:00"}
	rnd.HTML(200, "add", task)
}

func main() {
	TaskMap, _ = LoadState("/tmp/state.dmp")
	fmt.Println(TaskMap)
	for id, val := range TaskMap {
		if val.Running {
			//val.Ch = make(chan bool)
			go val.Start()
			TaskMap[id] = val
		}
	}
	m := martini.Classic()
	m.Use(render.Renderer(render.Options{
		Directory:  "templates",                // Specify what path to load the templates from.
		Layout:     "base",                     // Specify a layout template. Layouts can call {{ yield }} to render the current template.
		Extensions: []string{".tmpl", ".html"}, // Specify extensions to load for templates.
		Charset:    "UTF-8",                    // Sets encoding for json and html content-types. Default is "UTF-8".
		IndentJSON: true,                       // Output human readable JSON
	}))

	staticOptions := martini.StaticOptions{Prefix: "static"}
	m.Use(martini.Static("static", staticOptions))

	m.Get("/", indexHandler)
	m.Get("/tasks", indexHandler)
	m.Post("/store", storeHandler)
	m.Get("/edit/:id", editHandler)
	m.Get("/toggle/:id", toggleHandler)
	m.Get("/add", addHandler)
	m.Get("/delete/:id", deleteHandler)
	m.Run()

}
