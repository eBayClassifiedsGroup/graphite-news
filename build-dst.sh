#! /bin/bash

command -v $GOPATH/bin/go-bindata >/dev/null 2>&1 || { 
	echo "Installing go-bindata as it's a dependency"
	go get ./...
	go get github.com/jteeuwen/go-bindata/...
}

$GOPATH/bin/go-bindata -ignore=\\.swp$ index.html favicon.ico assets/...
go build
