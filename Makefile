#!/usr/bin/env make


GOPATH=$(shell pwd)/go
PACKAGE_NAME=github.com/jblawatt/gott/gott

tidy:
	go mod tidy

get:
	go get -v

build:
	go build



test:
	go test $(PACKAGE_NAME) -v
