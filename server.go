package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
)

type SMSPilotSettings struct {
	ApiKey string
	From   string
}

func (s *SMSPilotSettings) Store() error {
	fname := path.Join(Settings.SMSSettingsDir, "smspilot.db")
	b := new(bytes.Buffer)
	enc := gob.NewEncoder(b)
	err := enc.Encode(s)
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
func LoadSMSSettings() *SMSPilotSettings {
	fh, err := os.Open(path.Join(Settings.SMSSettingsDir, "smspilot.db"))
	t := SMSPilotSettings{}

	if err != nil {
		return &t
	}
	dec := gob.NewDecoder(fh)
	dec.Decode(&t)
	return &t
}

var timeout = time.Duration(2 * time.Second)

/*func PrepareRedis() (*redis.Client, error) {
	client, err := redis.DialTimeout("tcp", "127.0.0.1:6379", time.Duration(10)*time.Second)
	//defer client.Close()
	if err != nil {
		log.Println("failed to create the client", err)
		return client, err
	}
	client.Cmd("select", Settings.RedisDB)
	return client, nil

}
func MakeRedisCMD(cmd string, args ...interface{}) *redis.Reply {
	client, err := redis.DialTimeout("tcp", "127.0.0.1:6379", time.Duration(10)*time.Second)
	defer client.Close()
	if err != nil {
		log.Println("failed to create the client", err)
		return nil
	}
	client.Cmd("select", Settings.RedisDB)

	return client.Cmd(cmd, args)

}
func LoadRedisTasks() (statemap, error) {
	tm := make(statemap)
	//client, _ := PrepareRedis()

	res, err := MakeRedisCMD("hgetall", "gopoller/tasks").Hash()
	if err != nil {
		return nil, err
	}
	for _, val := range res {
		t := Task{}
		b := new(bytes.Buffer)
		b.Write([]byte(val))
		dec := gob.NewDecoder(b)
		err := dec.Decode(&t)
		if err != nil {
			continue
		}
		tm[t.Id] = &t
	}
	return tm, nil
}
*/
func LoadTasks(dir string) (statemap, error) {
	tm := make(statemap)
	files, _ := ioutil.ReadDir(dir)
	for _, f := range files {
		fh, err := os.Open(path.Join(dir, f.Name()))
		if err != nil {
			continue
		}
		t := Task{}
		dec := gob.NewDecoder(fh)
		err = dec.Decode(&t)
		defer fh.Close()
		if err == nil {
			tm[t.Id] = &t
		}

	}
	return tm, nil
}

func dialTimeout(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, timeout)
}

type SettingsType struct {
	RedisDB        int
	Dir            string
	SMSSettingsDir string
}

//var TaskMap = make(map[string]models.Task)
type statemap map[string]*Task

var TaskMap = make(statemap)
var Settings = SettingsType{}

func check(url string) (string, error) {
	transport := http.Transport{
		Dial: dialTimeout,
	}

	client := http.Client{
		Transport: &transport,
	}

	resp, err := client.Get(url)
	fmt.Println("check", url)
	var out string
	if err != nil {
		out = "host is unreachable"
	} else {

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			if string(body) != "OK" {
				out = "wrong answer"
				err = errors.New(out)
			} else {
				if len(body) > 100 {
					out = string(body[:100])

				} else {
					out = string(body)
				}
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
		task.Running = false
		task.Stop()
	} else {
		task.Running = true
		//task.Ch = make(chan bool)
		go task.Start()
	}
	TaskMap[id] = task
	task.Save()
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
	task.Delete()
	TaskMap.StoreState()
	rnd.Redirect("/")
}
func storesmsHandler(rnd render.Render, r *http.Request, params martini.Params) {
	apikey := r.FormValue("apikey")
	from := r.FormValue("from")
	t := SMSPilotSettings{From: from, ApiKey: apikey}
	t.Store()
	//MakeRedisCMD("hset", "gopoller/smspilot", "apikey", apikey)
	//MakeRedisCMD("hset", "gopoller/smspilot", "from", from)

	rnd.Redirect("/")
}
func storeHandler(rnd render.Render, r *http.Request, params martini.Params) {
	id := r.FormValue("id")
	url := r.FormValue("url")
	starttime := r.FormValue("starttime")
	stoptime := r.FormValue("stoptime")
	weekdays := r.Form["weekdays"]
	phone := r.FormValue("phone")
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
		task.Phone = phone
		for _, val := range weekdays {
			ival, _ := strconv.Atoi(val)
			tmp := task.Weekdays[ival]
			tmp.Enabled = true
			task.Weekdays[ival] = tmp
		}
		//task.Weekdays = weekdays
		task.Save()
		if task.Running {
			go task.Start()
		}
		TaskMap[id] = task
	} else {
		id = GenerateId()
		task := NewTask(id, url, period)
		task.Phone = phone
		for _, val := range weekdays {
			ival, _ := strconv.Atoi(val)
			tmp := task.Weekdays[ival]
			tmp.Enabled = true
			task.Weekdays[ival] = tmp
		}

		//task.Weekdays = weekdays
		task.Save()
		TaskMap[id] = task
	}
	TaskMap.StoreState()

	rnd.Redirect("/")
}

func addHandler(rnd render.Render) {
	id := ""
	task := NewTask(id, "", 30)
	//TaskMap[id] = task
	//task := Task{Period: 30, StartTime: "8:00", StopTime: "20:00"}
	rnd.HTML(200, "add", task)
}
func addSMSHandler(rnd render.Render) {
	//settings, _ := MakeRedisCMD("hgetall", "gopoller/smspilot").Hash()
	settings := LoadSMSSettings()

	rnd.HTML(200, "addsms", settings)
}
func main() {
	//TaskMap, _ = LoadState("/tmp/state.dmp")
	Settings.RedisDB = 4
	Settings.Dir = "gopollerdb/tasks"
	Settings.SMSSettingsDir = "gopollerdb/smssettings"
	_, err := os.Stat(Settings.Dir)
	if err != nil {
		os.MkdirAll(Settings.Dir, 0755)
	}
	_, err = os.Stat(Settings.SMSSettingsDir)
	if err != nil {
		os.MkdirAll(Settings.SMSSettingsDir, 0755)
	}

	TaskMap, _ = LoadTasks(Settings.Dir)
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

	staticOptions := martini.StaticOptions{Prefix: "static", SkipLogging: true}
	m.Use(martini.Static("static", staticOptions))

	m.Get("/", indexHandler)
	m.Get("/tasks", indexHandler)
	m.Post("/store", storeHandler)
	m.Post("/storesms", storesmsHandler)

	m.Get("/edit/:id", editHandler)
	m.Get("/toggle/:id", toggleHandler)
	m.Get("/add", addHandler)
	m.Get("/addsms", addSMSHandler)
	m.Get("/delete/:id", deleteHandler)
	log.Fatal(http.ListenAndServe("127.0.0.1:3000", m))

}
