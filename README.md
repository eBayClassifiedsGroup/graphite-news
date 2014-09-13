Graphite-News
=============

There are two leading theories as to why Graphite-News was created: people
generally had nothing better to do than wonder which new types of information
were being stored in their Graphite/Carbon databases. Or the second theory; I
needed a little pet project to try out some programming languages, etc. I will
leave it up to you.

Functionality
-------------
Graphite-News keeps tabs on which new data sources appear in your Graphite
storage so that you can see what new functionallity is being shipped inside
/your/ application. This is then exposed in a easy and simple web interface.

Technology
----------
Build with (or exists despite the following things being used):
 * Go (for tailing, server side state, webserver)
    * go-bindata (to keep everything in 1 file for easy deployment)
    * go-metrics (ofcourse we keep tabs on our own metrics as well)
 * jQuery
 * [Travis.ci](https://travis-ci.org/ojilles/graphite-news) for running all the unit tests: [![Build Status](https://travis-ci.org/ojilles/graphite-news.svg?branch=master)](https://travis-ci.org/ojilles/graphite-news)

