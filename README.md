Graphite-News
=============

There are two leading theories as to why Graphite-News was created: people
generally had nothing better to do than wonder which new types of information
were being stored in their Graphite/Carbon databases. Or the *second* theory; I
needed a little pet project to try out some programming languages, etc. I will
leave it up to you.

Functionality
-------------
Graphite-News keeps tabs on which new data sources appear in your Graphite
storage so that you can see what new functionality is being shipped inside
*your* application. This is then exposed in a easy and simple web interface.

Technology
----------
Build with (or exists despite the following things being used):
 * Go (for tailing, server side state, webserver)
    * [go-bindata](https://github.com/jteeuwen/go-bindata) (to keep 
      everything in 1 file for easy deployment)
    * [go-metrics](https://github.com/rcrowley/go-metrics) (of course 
      we keep tabs on our own metrics as well)
    * Tested on Go 1.3 and tip
 * jQuery
 * Bootstrap
 * [Travis.ci](https://travis-ci.org/ojilles/graphite-news) for running all the unit tests: [![Build Status](https://travis-ci.org/ojilles/graphite-news.svg?branch=master)](https://travis-ci.org/ojilles/graphite-news)

Installing
----------
Couple of options, but the easiest (assuming you have Go setup) is just running
the following:

    $ go get github.com/ojilles/graphite-news
    $ $GOPATH/bin/graphite-news -l $GOPATH/src/github.com/ojilles/graphite-news/creates.log

(That last command will get it up and running, but is obviously not how you
actually want to operate this piece of software.) If you have recently created new data sources, you will see those. If not, create a new one for testing purposes with:

    $ PORT=2003; SERVER=localhost; echo "local.random.diceroll 4 `date +%s`" | nc ${SERVER} ${PORT}

This should get you a `graphite-news` binary in `$GOPATH/bin`. Getting help gives you:

    $ graphite-news -h

    Usage: graphite-news [-i sec] [-p port] [-s graphite url] [-r] -l logfile
    
      -i=5000: Number of [ms] interval for Web UI's to update themselves. Clients only update 
               their config every 5min
      -l=[]: One or more locations of the Carbon logfiles we need to tail. 
             (F.ex. -l file1 -l file2 -l *.log)
      -p=2934: Port number the webserver will bind to (pick a free one please)
      -r=false: If set, report our own statistics every minute to a graphite host
      -rh="localhost:2003": Change the graphite host for pushing metrics towards
      -rp="graphite-news.metrics": Prepend all metric names with this string
      -s="http://localhost:8080": URL of the Graphite render API, no trailing 
             slash. Apple rendezvous domains do not work (like http://machine.local, use 
             IPs in that case)

The two important ones are the input (`-l` should point to the carbon logfile,
or whatever is storing the standard output of the carbon deamon) and the output
(`-s` the URL of the Graphite render API). Example:

    $ ~/graphite-news -l /opt/graphite/log/launchctl-carbon.stdout -s http://192.168.1.66:8080

Currently `-l` does not allow for globbing or multiple files in general.


Usage of Webinterface
---------------------
After starting `graphite-news`, it will tail through your logfile in search for
notifications that new data sources have been created and store those in
memory. If you point a browser to it (default port 2934, configurable through
`-p`) you are presented with this UI:

![Screenshot](https://raw.githubusercontent.com/ojilles/graphite-news/master/docs/images/screenshot-1.png)

Walkthrough:

 * Top left: `Server Connection` shows you if the client (this web app) is able
   to talk to the server side component.
 * Top right:
   * Shows the number of data sources in the server, bit useless at the moment
   * A button with which you can freeze your client with (e.g. no more data
     retrieval from the server)
 * The rest of the UI is a long list of new data sources found with some
   information on them. Interactions:
   * Click on a line, will open that up and show you the graph for that metric
   * Click on any other line, the previous one will close and the one belonging
     to the new line opens
   * Click on that graph, and it gets closed
 * Do nothing: the client will keep polling the server for more data sources
   and once found will automatically refresh and put that onto the page.

As you can see, very simple interaction model!

Reporting statistics to Graphite
--------------------------------
The go server process collects metrics about itself as well. These can be
reported back to graphite by using the `-r` flag, which is by default off. Once
enabled it will report all metrics to your Grahite server, by default located
at `localhost:2003` but can be overridden by using the `-rh` flag. Metrics by
default will be reported under `graphite-news.metrics` at the top root, which
can be altered by using the `-rp` flag. Most likely, this will incite an
Inception-level experience to the operator once he views the Webinterface.

As an example, here is a screenshot from the `json` HTTP end-point (which is
used to ferry metrics from server to client-side browser):

![metrics-screenshot](https://raw.githubusercontent.com/ojilles/graphite-news/master/docs/images/metrics-screenshot.png)

Couple of notes:

 * `JSON` end-point is requested by a client every 5 seconds. As you can see in
   the screenshot, the `GET.json.one-minute` metric shows ~0.2 (as there was
   one client connected). Same with the
`cactiStyle(scaleToSeconds(nonNegativeDerivative(graphite-news.metrics.GET.json.count),1))`.
 * All timing metrics are expressed as nanoseconds. To get back to the more
   often used milliseconds in the web-domain, scale by 0.000001. For example:
   `sortByName(cactiStyle(scale(graphite-news.metrics.GET.{json}.*-percentile,0.000001),"si"))`
 * `tail.input_lines.count` and `tail.datasources.count` will let you know how
   many log-lines have been parsed and how many data sources have been found
   respectively. Here is an example graph to illustrate:

![lines-screenshot](https://raw.githubusercontent.com/ojilles/graphite-news/master/docs/images/lines-screenshot.png)


To Compile
-----------
By default (`go build` or `go install`) the binary created does include all
static assets (such as javascript, css) needed for proper functioning, but
those are **not pulled from the files on disk**, but rather from `bindata.go`.
Regenerating `bindata.go` can be done with the provided `build-dst.sh` script.
This will result with all fresh assets in the binary (able to just `scp` to
another machine for example).  Effectively, `build-dst.sh` only does:

`go-bindata -ignore=\\.swp$ assets/...`

