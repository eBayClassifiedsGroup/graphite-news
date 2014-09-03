package main

import (
	"fmt"
	"github.com/ActiveState/tail"
	"github.com/op/go-logging"
	"net/http"
	"regexp"
	"strings"
)

func makeHandler(fn func(http.ResponseWriter, *http.Request, []string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, dss []string) {
	var log = logging.MustGetLogger("example")
	log.Info(fmt.Sprintf("Request for %s\n", r.URL.Path))
	fmt.Fprintf(w, "Hi there, I love %s %s!", r.URL.Path[1:], dss)
}

func tailLogfile(dss []string, c chan string) {
	var log = logging.MustGetLogger("example")

	//var dss []string
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
#http://stackoverflow.com/questions/18487923/golang-storing-caching-values-to-be-served-in-following-http-requests
	http.HandleFunc("/view/", makeHandler(viewHandler))
	go http.ListenAndServe(":2934", nil)
	go tailLogfile(dss, error_channel)

	log.Notice("Graphite News -- Showing which new metrics are available since 2014\n")

	// Wait for errors to appear then shut down
	log.Notice(<-error_channel)
}
