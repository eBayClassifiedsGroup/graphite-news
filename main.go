package main

// Implements a server that tails Graphite's logs
// finds out when new data sources become available
// and serves out a UI that lets you see/render those
// continuously.

// Written by J.A. Oldenbeuving / ojilles@gmail.com

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ActiveState/tail"
	"github.com/cespare/go-apachelog"
	"github.com/rcrowley/go-metrics"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type (

	// Structure of a single data source. Anything in capitals will
	// get marshalled over to any connecting browser
	Datasource struct {
		Name        string
		Create_date time.Time
		Params      string
	}

	// Holds the state (all newly detected data sources)
	state struct {
		*sync.RWMutex // inherits locking methods
		Vals          []Datasource
	}

	// Holds all configuration items for main. Anything with a capital
	// will get marshalled towards any browsers connecting
	configuration struct {
		JsonPullInterval int
		GraphiteURL      string
		ServerPort       int
		logfileLocation  loglocslice
	}

	// used for parsing Flags input params
	loglocslice []string
)

var (
	// declare a globally scoped State variable, otherwise
	// the request handlers can't get to it. If there is a better
	// way to do this, plmk.
	State = &state{&sync.RWMutex{}, []Datasource{}}

	// Instantiate struct to hold our configuration
	C = configuration{JsonPullInterval: 5000}
)

const (
	// Maximum number of data sources to hold in memory before
	// pruning out as new ones come in
	maxState    int = 100
	myLogFormat     = log.Ldate | log.Ltime

	staticAssetsURL = "/assets/"
)

func (i *loglocslice) String() string {
	return fmt.Sprintf("%v", *i)
}

// Used to do Flag parsing of unbounded number of input logfiles
func (i *loglocslice) Set(value string) error {
	*i = AppendIfMissing(*i, value)
	return nil
}

func init() {
	flag.IntVar(&C.JsonPullInterval, "i", 5000, "Number of [ms] interval for Web UI's to update themselves. Clients only update their config every 5min")
	flag.IntVar(&C.ServerPort, "p", 2934, "Port number the webserver will bind to (pick a free one please)")
	flag.StringVar(&C.GraphiteURL, "s", "http://localhost:8080", "URL of the Graphite render API, no trailing slash. Apple rendezvous domains do not work (like http://machine.local, use IPs in that case)")
	flag.Var(&C.logfileLocation, "l", "One or more locations of the Carbon logfiles we need to tail. (F.ex. -l file1 -l file2 -l *.log)")

	flag.Usage = func() {
		fmt.Printf("Usage: graphite-news [-i sec] [-p port] [-s graphite url] -l logfile \n\n")
		flag.PrintDefaults()
	}
}

func makeHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := metrics.GetOrRegisterTimer(fmt.Sprintf("%s%s", r.Method, r.URL.Path), metrics.DefaultRegistry)
		m.Time(func() {
			fn(w, r)
		})
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
	js, err := json.Marshal(C)
	if err != nil || len(js) < 1 {
		errorHandler(w, r, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func frontpageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	data, err := Asset("assets/index.html")
	if err != nil || len(data) == 0 || r.URL.String() != "/" {
		errorHandler(w, r, http.StatusNotFound)
		return
	}
	w.Write(data)
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	// URLs come in like "/assets/js/xxx", then get transformed to
	//                   "assets/static/js/xxx"
	// E.g. skip leading slash, inject static, where /assets/ is a constant
	filePath := fmt.Sprintf("%vstatic/%v",
		staticAssetsURL,
		r.RequestURI[len(staticAssetsURL):])[1:]
	tmp, err := Asset(filePath)
	if err != nil || len(tmp) == 0 {
		errorHandler(w, r, http.StatusNotFound)
		return
	}
	w.Write(tmp)
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	data, err := Asset("assets/favicon.ico")
	if err != nil {
		errorHandler(w, r, http.StatusNotFound)
		return
	}
	w.Write(data)
}

func errorHandler(w http.ResponseWriter, r *http.Request, status int) {
	w.WriteHeader(status)
	if status == http.StatusNotFound {
		fmt.Fprint(w, "<b>404</b>: Not Found")
	}
}

func parseTime(s string) time.Time {
	// 24/08/2014 20:59:54
	var t, _ = time.Parse("02/01/2006 15:04:05", s)
	return t
}

func addItemToState(ds Datasource) {
	var foundDuplicate = false
	l := log.New(os.Stdout, "tail	", myLogFormat)
	m_ds := metrics.GetOrRegisterCounter("tail/datasources", metrics.DefaultRegistry)

	if strings.HasSuffix(ds.Name, ".") {
		return
	}

	// Find out if we already have one with the same name, if
	// so skip it.
	State.RLock()

	// TODO: change to range
	for i := 0; i < len(State.Vals) && !foundDuplicate; i++ {
		if ds.Name == State.Vals[i].Name {
			foundDuplicate = true
		}
	}
	State.RUnlock()

	if !foundDuplicate {
		// We're writing to shared datastructures, grab a Write-lock
		State.Lock()
		defer State.Unlock()
		State.Vals = append(State.Vals, ds)
		defer m_ds.Inc(1)
		if len(State.Vals) > maxState {
			State.Vals = State.Vals[len(State.Vals)-maxState : maxState]
		}
		l.Printf("New datasource: %+v (total: %v)", ds.Name, len(State.Vals))
	}
}

func parseLine(line string) {
	m_lines := metrics.GetOrRegisterCounter("tail/input_lines", metrics.DefaultRegistry)
	var dataPath = regexp.MustCompile(`[a-zA-Z\:]*([0-9].*) :: \[creates\] creating database file .*/whisper/(.*)\.wsp (.*)`)
	m_lines.Inc(1)
	match := dataPath.FindStringSubmatch(line)
	if len(match) > 0 {
		ds := fmt.Sprintf("%s", strings.Replace(match[2], `/`, `.`, -1))
		tmp := Datasource{Name: ds, Create_date: parseTime(match[1]), Params: match[3]}
		addItemToState(tmp)
	}

}

func tailLogfile(c chan string, file string) {
	l := log.New(os.Stdout, "main	", myLogFormat)
	tc := tail.Config{Follow: true, ReOpen: true, MustExist: true}
	t, err := tail.TailFile(file, tc)
	if err == nil {
		l.Print(fmt.Sprintf("Tailing File:[%s]\n", file))
		for line := range t.Lines {
			parseLine(line.Text)
		}
	}
	c <- fmt.Sprintf("%s", err)
}

func tailLogfiles(c chan string) {
	var files []string

	// loop through the configured locations, and do filesystem
	// globbing, building up a new list. There is probably a special
	// place in POSIX-hell for me :-)
	for _, file := range C.logfileLocation {
		matches, _ := filepath.Glob(file)
		for _, match := range matches {
			files = AppendIfMissing(files, match)
		}
	}
	for _, file := range files {
		go tailLogfile(c, file)
	}
}

func AppendIfMissing(slice []string, i string) []string {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}

func main() {
	error_channel := make(chan string)
	l := log.New(os.Stdout, "main	", myLogFormat)
	flag.Parse()

	// grab any remaining arguments and pretend they belong
	// to -l :-) (Also solves the -l * case for example)
	for _, argument := range flag.Args() {
		C.logfileLocation = AppendIfMissing(C.logfileLocation, argument)
	}

	// Set up metrics registry
	//	go metrics.Log(
	//		metrics.DefaultRegistry,
	//		5e9, // Xe9 -> X seconds
	//		log.New(os.Stdout, "metrics	", myLogFormat))

	// Set up web handlers in goroutines
	mux := http.NewServeMux()
	mux.HandleFunc("/json/", makeHandler(jsonHandler))
	mux.HandleFunc("/stats/", makeHandler(statsHandler))
	mux.HandleFunc("/config/", makeHandler(configHandler))

	// These are all handled by the compiled in Assets
	mux.HandleFunc("/", makeHandler(frontpageHandler))
	mux.HandleFunc("/favicon.ico", makeHandler(faviconHandler))
	mux.HandleFunc(staticAssetsURL, makeHandler(staticHandler))

	// Add the logging handler for Apache Common-ish log output
	loggingHandler := apachelog.NewHandler(mux, os.Stdout)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%v", C.ServerPort),
		Handler: loggingHandler,
	}

	go server.ListenAndServe()

	go tailLogfiles(error_channel)

	l.Println("Graphite News -- Showing which new metrics are available since 2014\n")
	l.Println(fmt.Sprintf("Graphite News -- http://localhost:%v		:: Main User Interface", C.ServerPort))
	l.Println(fmt.Sprintf("Graphite News -- http://localhost:%v/config/	:: Internal configuration in JSON", C.ServerPort))
	l.Println(fmt.Sprintf("Graphite News -- http://localhost:%v/stats/	:: Internal Metrics in JSON", C.ServerPort))
	l.Println(fmt.Sprintf("Configuration: %+v", C))
	// Wait for errors to appear then shut down
	l.Println(<-error_channel)
}
