package main

import (
	"fmt"
	"github.com/ActiveState/tail"
	"github.com/op/go-logging"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

type state struct {
	*sync.RWMutex                   // inherits locking methods
	Vals          map[string]string // map ids to values
}

// declare a globally scoped State variable, otherwise
// the request handlers can't get to it. If there is a better
// way to do this, plmk.
var State = &state{&sync.RWMutex{}, map[string]string{}}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	var log = logging.MustGetLogger("example")
	log.Notice("viewHandler: acquiring read-lock")

	State.RLock()         // grab a lock, but then don't forget to
	defer State.RUnlock() // unlock it again once we're done

	log.Info(fmt.Sprintf("Request for %s\n", r.URL.Path))
	fmt.Fprintf(w, "Hi there, I love %s %s!", r.URL.Path[1:], State.Vals)
}

func addItemToState(line string) {
	var log = logging.MustGetLogger("example")
	log.Notice("addItemToState: acquiring write-lock")
	State.Lock()
	defer State.Unlock()
	State.Vals["bogus"] = line
	State.Vals["bogus2"] = line
}

func tailLogfile(dss []string, c chan string) {
	var log = logging.MustGetLogger("example")

	var dataPath = regexp.MustCompile(`\[creates\] creating (database) file .*/whisper/(.*)\.wsp`)
	t, err := tail.TailFile("./creates.log", tail.Config{Follow: true, ReOpen: true, MustExist: true})
	if err == nil {
		for line := range t.Lines {
			match := dataPath.FindStringSubmatch(line.Text)
			if len(match) > 0 {
				log.Info(line.Text)
				ds := fmt.Sprintf("%v", strings.Replace(match[2], `/`, `.`, -1))
				addItemToState(ds)
				dss = append(dss, ds)
				log.Notice(fmt.Sprintf("Found new datasource, total: %v, newly added: %s", len(dss), ds))
			}
		}
	}
	c <- fmt.Sprintf("%s", err)
}

func main() {
	error_channel := make(chan string)
	var dss []string
	var log = logging.MustGetLogger("example")
	var format = "%{color}%{time:15:04:05.000000} [%{pid}] ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}"

	// Setup logger https://github.com/op/go-logging/blob/master/examples/example.go
	logging.SetFormatter(logging.MustStringFormatter(format))
	log.Info("Graphite News -- Showing which new metrics are available since 2014\n")
	log.Notice("Graphite News -- Serving UI on: http://localhost:2934\n")

	// http://stackoverflow.com/questions/18487923/golang-storing-caching-values-to-be-served-in-following-http-requests

	http.HandleFunc("/view/", viewHandler)
	go http.ListenAndServe(":2934", nil)
	go tailLogfile(dss, error_channel)

	log.Notice("Graphite News -- Showing which new metrics are available since 2014\n")

	// Wait for errors to appear then shut down
	log.Error(<-error_channel)
}
