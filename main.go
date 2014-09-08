package main

import (
	"encoding/json"
	"fmt"
	"github.com/ActiveState/tail"
	"github.com/op/go-logging"
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

// declare a globally scoped State variable, otherwise
// the request handlers can't get to it. If there is a better
// way to do this, plmk.
var State = &state{&sync.RWMutex{}, []Datasource{}}

func makeHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r)
	}
}

func jsonHandler(w http.ResponseWriter, r *http.Request) {
	m := metrics.GetOrRegisterTimer("www/jsonHandler", metrics.DefaultRegistry)
	m.Time(func() {
		var log = logging.MustGetLogger("example")

		log.Notice("jsonHandler: acquiring read-lock")
		State.RLock() // grab a lock, but then don't forget to
		log.Notice("jsonHandler: got read-lock")
		defer State.RUnlock() // unlock it again once we're done
		defer log.Notice("jsonHandler: releaseing read-lock")

		log.Info(fmt.Sprintf("Request for %s\n", r.URL.Path))
		js, err := json.Marshal(State.Vals)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	})
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	m := metrics.GetOrRegisterTimer("www/statsHandler", metrics.DefaultRegistry)
	m.Time(func() {
		w.Header().Set("Content-Type", "application/json")

		metrics.WriteJSONOnce(metrics.DefaultRegistry, w)

	})
}
func parseTime(s string) time.Time {
	// 24/08/2014 20:59:54
	var t, _ = time.Parse("02/01/2006 15:04:05", s)
	return t
}

func addItemToState(ds Datasource) {
	m_ds := metrics.GetOrRegisterCounter("tail/datasources", metrics.DefaultRegistry)
	defer m_ds.Inc(1)

	var log = logging.MustGetLogger("example")
	log.Notice("addItemToState: acquiring write-lock")
	State.Lock()
	defer State.Unlock()
	defer log.Notice("addItemToState: released write-lock")
	log.Notice("addItemToState: got write-lock")
	State.Vals = append(State.Vals, ds)
}

func tailLogfile(c chan string) {
	m_lines := metrics.GetOrRegisterCounter("tail/input_lines", metrics.DefaultRegistry)

	var log = logging.MustGetLogger("example")

	var dataPath = regexp.MustCompile(`.*out:(.*) :: \[creates\] creating database file .*/whisper/(.*)\.wsp (.*)`)
	t, err := tail.TailFile("./creates.log", tail.Config{Follow: true, ReOpen: true, MustExist: true})
	if err == nil {
		for line := range t.Lines {
			m_lines.Inc(1)
			match := dataPath.FindStringSubmatch(line.Text)
			if len(match) > 0 {
				ds := fmt.Sprintf("%s", strings.Replace(match[2], `/`, `.`, -1))
				// log: 	  /opt/graphite/storage/whisper/big-imac-2011_local/collectd/memory/memory-inactive.wsp
				// real:          mac-mini-2014_local.collectd.memory.memory-active
				// found: 	  big-imac-2011_local.collectd.memory.memory-inactive
				tmp := Datasource{Name: ds, Create_date: parseTime(match[1]), Params: match[3]}
				addItemToState(tmp)
				log.Notice(fmt.Sprintf("Found new datasource, total: %v, newly added: %+v", len(State.Vals), tmp))
			}
		}
	}
	c <- fmt.Sprintf("%s", err)
}

func metricsRegister() {
	c := metrics.NewCounter()
	metrics.Register("foo", c)
	c.Inc(47)
}

func main() {
	error_channel := make(chan string)

	// Set up Logger
	// Setup logger https://github.com/op/go-logging/blob/master/examples/example.go
	//var log = logging.MustGetLogger("example")
	//var format = "%{color}%{time:15:04:05.000000} [%{pid}] ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}"
	var format = "%{color}%{time:15:04:05} [%{pid}] ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}"
	logging.SetFormatter(logging.MustStringFormatter(format))

	// Set up metrics registry
	go metrics.Log(
		metrics.DefaultRegistry,
		5e9, // Xe9 -> X seconds
		log.New(os.Stderr, "metrics ", log.Lmicroseconds),
	)

	// Set up web handlers in goroutines

	http.HandleFunc("/json/", makeHandler(jsonHandler))
	http.HandleFunc("/stats/", makeHandler(statsHandler))
	go http.ListenAndServe(":2934", nil)
	go tailLogfile(error_channel)

	//log.Notice("Graphite News -- Showing which new metrics are available since 2014\n")
	//log.Notice("Graphite News -- Serving UI on: http://localhost:2934\n")

	// Wait for errors to appear then shut down
	fmt.Println(<-error_channel)
}
