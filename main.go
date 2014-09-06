package main

import (
	"encoding/json"
	"fmt"
	"github.com/ActiveState/tail"
	"github.com/op/go-logging"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

type Datasource struct {
	Name        string
	Create_date string // for now, obv needs to be different
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

func viewHandler(w http.ResponseWriter, r *http.Request) {
	var log = logging.MustGetLogger("example")

	log.Notice("viewHandler: acquiring read-lock")
	State.RLock()         // grab a lock, but then don't forget to
	defer State.RUnlock() // unlock it again once we're done

	log.Info(fmt.Sprintf("Request for %s\n", r.URL.Path))
	js, err := json.Marshal(State.Vals)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func addItemToState(ds Datasource) {
	var log = logging.MustGetLogger("example")
	log.Notice("addItemToState: acquiring write-lock")
	State.Lock()
	defer State.Unlock()
	defer log.Notice("addItemToState: released write-lock")
	State.Vals = append(State.Vals, ds)
}

func tailLogfile(c chan string) {
	var log = logging.MustGetLogger("example")

	var dataPath = regexp.MustCompile(`.*out:(.*) :: \[creates\] creating database file .*/whisper/(.*)\.wsp (.*)`)
	t, err := tail.TailFile("./creates.log", tail.Config{Follow: true, ReOpen: true, MustExist: true})
	if err == nil {
		for line := range t.Lines {
			match := dataPath.FindStringSubmatch(line.Text)
			if len(match) > 0 {
				ds := fmt.Sprintf("%s", strings.Replace(match[2], `/`, `.`, -1))
				tmp := Datasource{Name: ds, Create_date: match[1], Params: match[3] }
				addItemToState(tmp)
				log.Notice(fmt.Sprintf("Found new datasource, total: %v, newly added: %v", len(State.Vals), tmp))
			}
		}
	}
	c <- fmt.Sprintf("%s", err)
}

func main() {
	error_channel := make(chan string)
	var log = logging.MustGetLogger("example")
	var format = "%{color}%{time:15:04:05.000000} [%{pid}] ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}"

	// Setup logger https://github.com/op/go-logging/blob/master/examples/example.go
	logging.SetFormatter(logging.MustStringFormatter(format))
	log.Info("Graphite News -- Showing which new metrics are available since 2014\n")
	log.Notice("Graphite News -- Serving UI on: http://localhost:2934\n")

	// http://stackoverflow.com/questions/18487923/golang-storing-caching-values-to-be-served-in-following-http-requests

	http.HandleFunc("/view/", viewHandler)
	go http.ListenAndServe(":2934", nil)
	go tailLogfile(error_channel)

	log.Notice("Graphite News -- Showing which new metrics are available since 2014\n")

	// Wait for errors to appear then shut down
	log.Error(<-error_channel)
}
