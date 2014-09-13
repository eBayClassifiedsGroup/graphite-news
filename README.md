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
    Usage of graphite-news:

      -i=5000: Number of [ms] interval for Web UI's to update themselves. Clients only update 
         their config every 5min
      -l="creates.log": Location of the Carbon logfiles we need to tail
      -p=2934: Port number the webserver will bind to (pick a free one please)
      -s="http://localhost:8080": URL of the Graphite render API, no trailing slash. Apple 
         rendezvous domains do not work (like http://machine.local, use IPs in that case)

The two important ones are the input (`-l` should point to the carbon logfile,
or whatever is storing the standard output of the carbon deamon) and the output
(`-s` the URL of the Graphite render API). Example:

    $ ~/graphite-news -l /opt/graphite/log/launchctl-carbon.stdout -s http://192.168.1.66:8080

Currently `-l` does not allow for globbing or multiple files in general.

Usage
-----
After starting `graphite-news`, it will tail through your logfile in search for
notifications that new data sources have been created and store those in
memory. If you point a browser to it (default port 2934, configurable through
`-p`) you are presented with this UI:

![Screenshot](https://raw.githubusercontent.com/ojilles/graphite-news/master/screenshot-1.png)

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

To Compile
-----------
By default (`go build` or `go install`) the binary created does include all
static assets (such as javascript, css) needed for proper functioning, but
those are **not pulled from the files on disk**, but rather from `bindata.go`.
Regenerating `bindata.go` can be done with the provided `build-dst.sh` script.
This will result with all fresh assets in the binary (able to just `scp` to
another machine for example).  Effectively, `build-dst.sh` only does:

`go-bindata -ignore=\\.swp$ index.html favicon.ico assets/...`

