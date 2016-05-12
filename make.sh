#!/usr/bin/env bash

if [ ! -f make.sh ]; then
	echo 'make.sh must be run within its container folder' 1>&2
	exit 1
fi

start_time=`date +%s`
echo "waiting for compiling..."

go install DorMediaServer

end_time=`date +%s`

echo "total: " $[$end_time - $start_time] "s"

if [ "$1" = "fmt" ]; then
    gofmt -w src
fi

if [ "$1" = "test" ]; then
    go test src/tests
fi
