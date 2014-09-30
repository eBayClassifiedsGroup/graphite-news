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
	"net"
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
		Name        string    // bla.te.jfwoiejf.1MinuteRate, etc
		Create_date time.Time // Holds timestamp of when DS got created
		Params      string    // Holds things like retention schema's, etc
		filename    string    // /opt/graphite/whisper/etc
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

		// If set, allow server to delete data sources that were encountered (not
		// random ones, only the ones in the State (e.g. the last maxState # of items)
		AllowDsDeletes bool

		// These are used for reporting Graphite-news' own
		// stats to a Graphite server. If that server is being monitored
		// by Graphite-news, you've recreated Inception!
		reporterGraphiteEnabled bool
		reporterGraphiteHost    string
		reporterGraphitePrep    string
	}

	// used for parsing Flags input params
	loglocslice []string
)

var (
	// Will hold version information from https://github.com/laher/goxc
	VERSION    string
	BUILD_DATE string

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
	if len(VERSION) == 0 {
		VERSION = "non-packaged"
		BUILD_DATE = "now"
	}

	flag.IntVar(&C.JsonPullInterval, "i", 5000, "Number of [ms] interval for Web UI's to update themselves. Clients only update their config every 5min")
	flag.IntVar(&C.ServerPort, "p", 2934, "Port number the webserver will bind to (pick a free one please)")
	flag.StringVar(&C.GraphiteURL, "s", "http://localhost:8080", "URL of the Graphite render API, no trailing slash. Apple rendezvous domains do not work (like http://machine.local, use IPs in that case)")
	flag.Var(&C.logfileLocation, "l", "One or more locations of the Carbon logfiles we need to tail. (F.ex. -l file1 -l file2 -l *.log)")
	flag.BoolVar(&C.AllowDsDeletes, "d", false, "If set, allow clients to delete recently created data sources")
	flag.BoolVar(&C.reporterGraphiteEnabled, "r", false, "If set, report our own statistics every minute to a graphite host")
	flag.StringVar(&C.reporterGraphiteHost, "rh", "localhost:2003", "Change the graphite host for pushing metrics towards")
	flag.StringVar(&C.reporterGraphitePrep, "rp", "graphite-news.metrics", "Prepend all metric names with this string")

	flag.Usage = func() {
		fmt.Printf("Usage: graphite-news [-i sec] [-p port] [-s graphite url] [-r] [-d] -l logfile \n")
		fmt.Printf("Version: %v (Compiled at %v). Code over at: https://github.com/ojilles/graphite-news/\n\n", VERSION, BUILD_DATE)
		flag.PrintDefaults()
	}
}

func makeHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// For metric names, escape any natural dots in the name,
		// then replace any URL slashes with a dot (seems same semantics to me)
		// and make sure we dont have any starting or ending with a dot.
		metricUrl := strings.Replace(r.URL.Path, ".", "_", -1)
		metricUrl = strings.Replace(metricUrl, "/", ".", -1)
		metricUrl = strings.Trim(metricUrl, ".")
		if len(metricUrl) == 0 {
			metricUrl = "index_html"
		}

		metricName := fmt.Sprintf("%s.%s", r.Method, metricUrl)
		aggMetricName := fmt.Sprintf("%s.__all_reqs", r.Method)
		m := metrics.GetOrRegisterTimer(metricName, metrics.DefaultRegistry)
		mGet := metrics.GetOrRegisterTimer(aggMetricName, metrics.DefaultRegistry)
		mGet.Time(func() {
			m.Time(func() {
				fn(w, r)
			})
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

func deleteFile(dsFilename string) (err bool) {
	l := log.New(os.Stdout, "main	", myLogFormat)
	m := metrics.GetOrRegisterCounter("deletes", metrics.DefaultRegistry)

	if len(dsFilename) < 1 {
		l.Printf("deleteFile called, ignoring b/c/o unlikely filename: %v", dsFilename)
		return false
	}

	if _, err := os.Stat(dsFilename); os.IsNotExist(err) {
		l.Printf("deleteFile called but no such file: %s", dsFilename)
		return false
	}

	removeErr := os.Remove(dsFilename)
	if removeErr != nil {
		l.Printf("deleteFile called but failed os.Remove call: %s", dsFilename)
		return false
	}

	l.Printf("deleteFile called and succeeded: %s", dsFilename)
	m.Inc(1)
	return true
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	l := log.New(os.Stdout, "main	", myLogFormat)

	if r.Method != "POST" {
		l.Printf("DELETE called, ignoring b/c was not a post: %v:%v\n", r.Method, r.URL)
		// Only allow POSTs to delete DS's
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	r.ParseForm()
	dsName := r.PostFormValue("datasourcename")
	ds := getDSbyName(dsName)
	Success := false

	if (len(ds.Name) > 0) && (len(ds.filename) > 0) {
		Success = deleteFile(ds.filename)
	}

	if Success == true {
		_ = deleteDSbyName(dsName)
		w.Write(nil)
	} else {
		http.Error(w, "", http.StatusInternalServerError)
	}

	l.Printf("DELETE called for '%v' (filename: %v) with result: '%v'",
		dsName, ds.filename, Success)
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
	m_ds := metrics.GetOrRegisterCounter("tail.datasources", metrics.DefaultRegistry)

	if strings.HasSuffix(ds.Name, ".") {
		return
	}

	// Find out if we already have one with the same name, if
	// so skip it.
	State.RLock()

	for _, item := range State.Vals {
		if !foundDuplicate && ds.Name == item.Name {
			foundDuplicate = true
		}
	}

	// Can not defer this one, as it would prevent any write lock below from
	// getting aqcuired succesfully (race cond.)
	State.RUnlock()

	if !foundDuplicate {
		// We're writing to shared datastructures, grab a Write-lock
		State.Lock()
		defer State.Unlock()
		State.Vals = append(State.Vals, ds)

		if len(State.Vals) > maxState {
			State.Vals = State.Vals[len(State.Vals)-maxState : len(State.Vals)]
		}

		l.Printf("New datasource: %+v (total: %v)", ds.Name, len(State.Vals))
		defer m_ds.Inc(1)
	}
}

func parseLine(line string) {
	m_lines := metrics.GetOrRegisterCounter("tail.input_lines", metrics.DefaultRegistry)
	var dataPath = regexp.MustCompile(`[a-zA-Z\:]*([0-9].*) :: \[creates\] creating database file (.*/whisper/(.*)\.wsp) (.*)`)
	m_lines.Inc(1)
	match := dataPath.FindStringSubmatch(line)
	if len(match) > 0 {
		ds := fmt.Sprintf("%s", strings.Replace(match[3], `/`, `.`, -1))
		tmp := Datasource{Name: ds, Create_date: parseTime(match[1]), Params: match[4], filename: match[2]}
		addItemToState(tmp)
	}

}

func deleteDSbyName(dsName string) bool {
	if len(getDSbyName(dsName).Name) == 0 {
		return false
	}

	State.Lock()
	defer State.Unlock()

	for i, ds_tmp := range State.Vals {
		if ds_tmp.Name == dsName {
			State.Vals = append(State.Vals[:i], State.Vals[i+1:]...)
			return true
		}
	}
	return false
}

func getDSbyName(dsName string) Datasource {
	State.RLock()
	defer State.RUnlock()

	for _, ds_tmp := range State.Vals {
		if ds_tmp.Name == dsName {
			return ds_tmp
		}
	}
	x := Datasource{}
	return x
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

func reportMetrics() {
	// Set up metrics registry
	if C.reporterGraphiteEnabled {
		addr, _ := net.ResolveTCPAddr("tcp", C.reporterGraphiteHost)
		go metrics.Graphite(metrics.DefaultRegistry, 60e9, C.reporterGraphitePrep, addr)
	}
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

	// Set up web handlers in goroutines
	mux := http.NewServeMux()
	mux.HandleFunc("/json/", makeHandler(jsonHandler))
	mux.HandleFunc("/stats/", makeHandler(statsHandler))
	mux.HandleFunc("/config/", makeHandler(configHandler))
	mux.HandleFunc("/delete/", makeHandler(deleteHandler))

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
	go reportMetrics()

	l.Println("Graphite News -- Showing which new metrics are available since 2014")
	l.Println(fmt.Sprintf("Version: %v (Compiled at %v). Code over at: https://github.com/ojilles/graphite-news/", VERSION, BUILD_DATE))
	l.Println(fmt.Sprintf("Graphite News -- http://localhost:%v		:: Main User Interface", C.ServerPort))
	l.Println(fmt.Sprintf("Graphite News -- http://localhost:%v/config/	:: Internal configuration in JSON", C.ServerPort))
	l.Println(fmt.Sprintf("Graphite News -- http://localhost:%v/stats/	:: Internal Metrics in JSON", C.ServerPort))
	l.Println(fmt.Sprintf("Graphite News -- http://localhost:%v/json/	:: JSON dump of new graphite data sources", C.ServerPort))
	l.Println(fmt.Sprintf("Configuration: %+v", C))
	// Wait for errors to appear then shut down
	l.Println(<-error_channel)
}
