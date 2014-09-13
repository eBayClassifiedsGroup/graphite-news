#! /bin/bash

if hash gdate 2>/dev/null; then
	echo "Installing go-bindata as it's a dependency"
	go get github.com/jteeuwen/go-bindata/...
else
	go-bindata index.html assets/...
	go build
fi
