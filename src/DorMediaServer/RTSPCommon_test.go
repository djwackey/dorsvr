package main

import (
	"fmt"
	"testing"
)

var (
	OPTIONS_COMMAND = "OPTIONS rtsp://172.22.0.172/123.ts RTSP/1.0\r\n" +
		"CSeq: 2\r\n" +
		"User-Agent: LibVLC/2.1.2 (LIVE555 Streaming Media v2013.12.05)\r\n\r\n"

	DESCRIPTION_COMMAND = "DESCRIBE rtsp://192.168.1.103/live1 RTSP/1.0\r\n" +
		"CSeq: 3\r\n" +
		"User-Agent: LibVLC/2.1.5 (LIVE555 Streaming Media v2014.05.27)\r\n" +
		"Accept: application/sdp\r\n\r\n"

	PLAY_COMMAND = ""

	TEARDOWN_COMMAND = ""
)

func Test_ParseRTSPRequestString(t *testing.T) {
	reqInfo, result := ParseRTSPRequestString([]byte(OPTIONS_COMMAND), len(OPTIONS_COMMAND))
	if result {
		fmt.Println("cmdName:", reqInfo.cmdName)
		fmt.Println("CSeq:", reqInfo.cseq)
		fmt.Println("sessionIdStr:", reqInfo.sessionIdStr)
		t.Log("ok")
	} else {
	}
}

func Test_parseCommandName(t *testing.T) {
	cmdName, result := parseCommandName(OPTIONS_COMMAND)
	if result {
		fmt.Println(cmdName, result)
		t.Log("ok")
	} else {
		t.Error("failed")
	}
}
