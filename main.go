package main

// Implements a server that tails Graphite's logs
// finds out when new data sources become available
// and serves out a UI that lets you see/render those
// continiously.

// Written by J.A. Oldenbeuving / ojilles@gmail.com

import (
	"encoding/json"
	"fmt"
	"github.com/ActiveState/tail"
	"github.com/rcrowley/go-metrics"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"flag"
)

type Datasource struct {
	Name        string
	Create_date time.Time
	Params      string
}

type state struct {
	*sync.RWMutex // inherits locking methods
	Vals          []Datasource
}

type configuration struct {
	JsonPullInterval	int
	GraphiteURL		string
	ServerPort		int
	logfileLocation		string
}

const myLogFormat = log.Ldate | log.Ltime

// declare a globally scoped State variable, otherwise
// the request handlers can't get to it. If there is a better
// way to do this, plmk.
var State = &state{&sync.RWMutex{}, []Datasource{}}

// Instantiate struct to hold our configuration
var C = configuration{JsonPullInterval:5000, GraphiteURL:"asdfsdfadsf"};


func init() {
	flag.IntVar(&C.JsonPullInterval, "i", 5000,"Number of [ms] interval for Web UI's to update themselves. Clients only update their config every 5min")
	flag.IntVar(&C.ServerPort, "p", 2934, "Port number the webserver will bind to (pick a free one please)")
	flag.StringVar(&C.logfileLocation, "l", "creates.log", "Location of the Carbon logfiles we need to tail")
}


func makeHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := log.New(os.Stdout, "http	", myLogFormat)

		m := metrics.GetOrRegisterTimer(fmt.Sprintf("%s%s", r.Method, r.URL.Path), metrics.DefaultRegistry)
		m.Time(func() {
			fn(w, r)
		})
		l.Printf("Request: %v %v %v %v", r.Method, r.URL, r.RemoteAddr, m)
		//l.Printf("Request: %v %v %v %v", r.Method, r.URL, r.RemoteAddr, r.Header["User-Agent"])
	}
}

func jsonHandler(w http.ResponseWriter, r *http.Request) {
	State.RLock() // grab a lock, but then don't forget to
	js, err := json.Marshal(State.Vals)
	State.RUnlock() // unlock it again once we're done

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	metrics.WriteJSONOnce(metrics.DefaultRegistry, w)
}

func configHandler(w http.ResponseWriter, r *http.Request) {
	js, _ := json.Marshal(C)
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func frontpageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	bs, err := ioutil.ReadFile("index.html")
	if err != nil {
		bs = []byte("<b>Fatal Error</b>: index.html file not found.")
	}
	w.Write(bs)
}

func parseTime(s string) time.Time {
	// 24/08/2014 20:59:54
	var t, _ = time.Parse("02/01/2006 15:04:05", s)
	return t
}

func addItemToState(ds Datasource) {
	l := log.New(os.Stdout, "tail	", myLogFormat)
	m_ds := metrics.GetOrRegisterCounter("tail/datasources", metrics.DefaultRegistry)
	defer m_ds.Inc(1)

	// We're writing to shared datastructures, grab a Write-lock
	State.Lock()
	defer State.Unlock()
// TODO: find out if we have  already exactly this DS in State, then don't add it again.
	State.Vals = append(State.Vals, ds)
	l.Printf("New datasource: %+v (total: %v)", ds.Name, len(State.Vals))
}

func tailLogfile(c chan string) {
	m_lines := metrics.GetOrRegisterCounter("tail/input_lines", metrics.DefaultRegistry)

	var dataPath = regexp.MustCompile(`.*out:(.*) :: \[creates\] creating database file .*/whisper/(.*)\.wsp (.*)`)
	t, err := tail.TailFile(C.logfileLocation, tail.Config{Follow: true, ReOpen: true, MustExist: true})
	if err == nil {
		for line := range t.Lines {
			m_lines.Inc(1)
			match := dataPath.FindStringSubmatch(line.Text)
			if len(match) > 0 {
				ds := fmt.Sprintf("%s", strings.Replace(match[2], `/`, `.`, -1))
				tmp := Datasource{Name: ds, Create_date: parseTime(match[1]), Params: match[3]}
				addItemToState(tmp)
			}
		}
	}
	c <- fmt.Sprintf("%s", err)
}

func main() {
	error_channel := make(chan string)
	l := log.New(os.Stdout, "main	", myLogFormat)
	flag.Parse()

	// Set up metrics registry
	//	go metrics.Log(
	//		metrics.DefaultRegistry,
	//		5e9, // Xe9 -> X seconds
	//		log.New(os.Stdout, "metrics	", myLogFormat))

	// Set up web handlers in goroutines
	http.HandleFunc("/", makeHandler(frontpageHandler))
	http.HandleFunc("/json/", makeHandler(jsonHandler))
	http.HandleFunc("/stats/", makeHandler(statsHandler))
	http.HandleFunc("/config/", makeHandler(configHandler))

	http.Handle("/assets/",
		http.StripPrefix("/assets/",
			http.FileServer(http.Dir("./assets"))))
	go http.ListenAndServe(fmt.Sprintf(":%v",C.ServerPort), nil)
	go tailLogfile(error_channel)

	l.Println("Graphite News -- Showing which new metrics are available since 2014\n")
	l.Println(fmt.Sprintf("Graphite News -- http://localhost:%v		:: Main User Interface",C.ServerPort))
	l.Println(fmt.Sprintf("Graphite News -- http://localhost:%v/config/	:: Internal configuration in JSON",C.ServerPort))
	l.Println(fmt.Sprintf("Graphite News -- http://localhost:%v/stats/	:: Internal Metrics in JSON",C.ServerPort))
	l.Println(fmt.Sprintf("Configuration: %+v", C))
	// Wait for errors to appear then shut down
	l.Println(<-error_channel)
}
