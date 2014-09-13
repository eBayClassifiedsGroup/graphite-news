#! /bin/bash

if hash gdate 2>/dev/null; then
	echo "Installing go-bindata as it's a dependency"
	go get github.com/jteeuwen/go-bindata/...
else
	go-bindata -ignore=\\.swp$ index.html favicon.ico assets/...
	go build
fi
