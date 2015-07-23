package main

import (
	"fmt"
	"time"
	//. "groupsock"
	//. "include"
	//. "liveMedia"
)

var counter int
var (
	OPTIONS_COMMAND = "OPTIONS rtsp://172.22.0.172/123.ts RTSP/1.0\r\n" +
		"CSeq: 1\r\n" +
		"User-Agent: LibVLC/2.1.2 (LIVE555 Streaming Media v2013.12.05)\r\n\r\n"

	DESCRIPTION_COMMAND = "DESCRIBE rtsp://192.168.1.103/live1.264 RTSP/1.0\r\n" +
		"CSeq: 2\r\n" +
		"User-Agent: LibVLC/2.1.5 (LIVE555 Streaming Media v2014.05.27)\r\n" +
		"Accept: application/sdp\r\n\r\n"

	PLAY_COMMAND = ""

	TEARDOWN_COMMAND = ""
)

func livenessTimeoutTask(millisec time.Duration) {
	tm := time.NewTimer(time.Millisecond * millisec)

	for {
		select {
		case <-tm.C:
			tm.Reset(time.Millisecond * millisec)
			counter++
			fmt.Println("livenessTimeoutTask", counter)
			if counter > 10 {
				tm.Stop()
			}
		}
	}
}

func test() {
	for {
		time.Sleep(time.Second * 3)
		fmt.Println("test")
	}
}

func parse(reqStr string, reqStrSize int) bool {
	cmdName := make([]byte, 10)

	// Read everything up to the first space as the command name:
	var parseSucceeded bool
	i := 0
	for i = 0; i < reqStrSize; i++ {
		c := string(reqStr[i])
		if c == " " || c == "\t" {
			parseSucceeded = true
			break
		}
		cmdName[i] = reqStr[i]
	}

	if !parseSucceeded {
		return false
	}

	// skip over any additional white space
	j := i + 1
	for ; j < reqStrSize; j++ {
		c := string(reqStr[i])
		if c != " " && c != "\t" {
			break
		}
	}

	// Look for "CSeq:"

	// Look for "Session:"

	// Also: Look for "Content-Length:" (optional, case insensitive)

	fmt.Println(string(cmdName))
	return true
}

func main() {
	parse(DESCRIPTION_COMMAND, len(DESCRIPTION_COMMAND))
	/*
		for i := 0; i < 10; i++ {
			fmt.Println(i)
		}

		n := 10

		n++
		fmt.Println(n)
	*/

	//go livenessTimeoutTask(1000)
	//go test()
	//select {}
}
