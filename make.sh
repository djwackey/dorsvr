#!/usr/bin/env bash
if [ ! -f make.sh ]; then
	echo 'make.sh must be run within its container folder' 1>&2
	exit 1
fi


go install DorMediaServer


if [ "$1" = "fmt" ]; then
    gofmt -w src
fi

if [ "$1" = "test" ]; then
    go test src/tests
fi
