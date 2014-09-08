package main

import (
	"encoding/json"
	"fmt"
	"github.com/ActiveState/tail"
	"github.com/rcrowley/go-metrics"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
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

const myLogFormat = log.Ldate | log.Ltime

// declare a globally scoped State variable, otherwise
// the request handlers can't get to it. If there is a better
// way to do this, plmk.
var State = &state{&sync.RWMutex{}, []Datasource{}}

func makeHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := log.New(os.Stdout, "http	", myLogFormat)
		l.Printf("Request: %+v %+v", r.URL, r)
		m := metrics.GetOrRegisterTimer(fmt.Sprintf("%s%s", r.Method, r.URL.Path), metrics.DefaultRegistry)
		m.Time(func() {
			fn(w, r)
		})
	}
}

func jsonHandler(w http.ResponseWriter, r *http.Request) {
	l := log.New(os.Stdout, "http	", myLogFormat)

	State.RLock() // grab a lock, but then don't forget to
	defer State.RUnlock() // unlock it again once we're done

	l.Printf("Request for %s\n", r.URL.Path)
	js, err := json.Marshal(State.Vals)

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

	State.Vals = append(State.Vals, ds)
	l.Printf("New datasource: %+v (total: %v)", ds.Name, len(State.Vals))
}

func tailLogfile(c chan string) {
	m_lines := metrics.GetOrRegisterCounter("tail/input_lines", metrics.DefaultRegistry)

	var dataPath = regexp.MustCompile(`.*out:(.*) :: \[creates\] creating database file .*/whisper/(.*)\.wsp (.*)`)
	t, err := tail.TailFile("./creates.log", tail.Config{Follow: true, ReOpen: true, MustExist: true})
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

	// Set up metrics registry
	go metrics.Log(
		metrics.DefaultRegistry,
		5e9, // Xe9 -> X seconds
		log.New(os.Stdout, "metrics	", myLogFormat))

	// Set up web handlers in goroutines

	http.HandleFunc("/json/", makeHandler(jsonHandler))
	http.HandleFunc("/stats/", makeHandler(statsHandler))
	go http.ListenAndServe(":2934", nil)
	go tailLogfile(error_channel)

	l.Println("Graphite News -- Showing which new metrics are available since 2014\n")
	l.Println("Graphite News -- Serving UI on: http://localhost:2934\n")

	// Wait for errors to appear then shut down
	l.Println(<-error_channel)
}
