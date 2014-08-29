package main

import (
	"fmt"
	"github.com/ActiveState/tail"
	"github.com/op/go-logging"
	"net/http"
	"regexp"
	"strings"
)

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, "sadfds")
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	var log = logging.MustGetLogger("example")
	log.Info(fmt.Sprintf("Request for %s\n", r.URL.Path))
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])

}

func main() {

	// Setup logger https://github.com/op/go-logging/blob/master/examples/example.go
	var log = logging.MustGetLogger("example")

	var dss []string

	// Example format string. Everything except the message has a custom color
	// which is dependent on the log level. Many fields have a custom output
	// formatting too, eg. the time returns the hour down to the milli second.
	var format = "%{color}%{time:15:04:05.000000} [%{pid}] ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}"
	logging.SetFormatter(logging.MustStringFormatter(format))

	log.Info("Graphite News -- Showing which new metrics are available since 2014\n")
	log.Notice("Graphite News -- Serving UI on: http://localhost:2934\n")

	http.HandleFunc("/view/", makeHandler(viewHandler))
	go http.ListenAndServe(":2934", nil)

	log.Notice("Graphite News -- Showing which new metrics are available since 2014\n")
	var dataPath = regexp.MustCompile(`\[creates\] creating (database) file .*/whisper/(.*)\.wsp`)

	t, err := tail.TailFile("./creates.log", tail.Config{Follow: true, ReOpen: true, MustExist: true})
	if err == nil {
		for line := range t.Lines {
			match := dataPath.FindStringSubmatch(line.Text)
			//	log.Info(match)
			if len(match) > 0 {
				log.Info(line.Text)
				ds := fmt.Sprintf("%v", strings.Replace(match[2], `/`, `.`, -1))
				dss = append(dss, ds)
				log.Notice(fmt.Sprintf("Found new datasource, total: %v, newly added: %s", len(dss), ds))
			}
		}
	}
	fmt.Println(err)
}
