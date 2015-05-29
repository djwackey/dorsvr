package main

import (
	"fmt"
	"strings"
	"time"
)

// Handler routines for specific RTSP commands:
var allowedCommandNames [9]string = [9]string{"OPTIONS", "DESCRIBE", "SETUP", "TEARDOWN", "PLAY", "PAUSE", "RECORD", "GET_PARAMETER", "SET_PARAMETER"}

type RTSPRequestInfo struct {
	cmdName string
}

func ParseRTSPRequestString(buf []byte, len int) (*RTSPRequestInfo, bool) {
	result := true
	var cmdIndex int
	switch {
	case strings.HasPrefix(string(buf), allowedCommandNames[0]):
		cmdIndex = 0
	case strings.HasPrefix(string(buf), allowedCommandNames[1]):
		cmdIndex = 1
	case strings.HasPrefix(string(buf), allowedCommandNames[2]):
		cmdIndex = 2
	case strings.HasPrefix(string(buf), allowedCommandNames[3]):
		cmdIndex = 3
	case strings.HasPrefix(string(buf), allowedCommandNames[4]):
		cmdIndex = 4
	case strings.HasPrefix(string(buf), allowedCommandNames[5]):
		cmdIndex = 5
	case strings.HasPrefix(string(buf), allowedCommandNames[6]):
		cmdIndex = 6
	case strings.HasPrefix(string(buf), allowedCommandNames[7]):
		cmdIndex = 7
	case strings.HasPrefix(string(buf), allowedCommandNames[8]):
		cmdIndex = 8
	default:
	}

	return &RTSPRequestInfo{cmdName: allowedCommandNames[cmdIndex]}, result
}

// A "Date:" header that can be used in a RTSP (or HTTP) response
func DateHeader() string {
	return fmt.Sprintf("Date: %s\r\n", time.Now())
}
