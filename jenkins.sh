#!/bin/bash


type glide 2> /dev/null || /bin/sh -c "export GOPATH=$GOROOT && curl -k https://glide.sh/get | sh"
type go-junit-report 2> /dev/null || go get -u github.com/jstemmer/go-junit-report
type gocov 2> /dev/null || go get github.com/axw/gocov/gocov
type gocov-xml 2> /dev/null || go get github.com/AlekSi/gocov-xml
type gometalinter 2> /dev/null || go get -u gopkg.in/alecthomas/gometalinter && gometalinter -i


#Mac hack
#GOPATH=$GOROOT glide install

export JUNIT_REPORT=TEST-junit.xml
export COBERTURA_REPORT=coverage.xml
export CHECKSTYLE_REPORT=checkstyle.xml

# Go get is not the best way of installing.... :/
export PATH=$PATH:$HOME/go/bin

# The insanely slow compilel within docker in Mac makes me want to cheat
#make clean test
