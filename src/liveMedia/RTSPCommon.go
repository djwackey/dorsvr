package liveMedia

import (
	"fmt"
	"strings"
	"time"
)

const MAX_COMMAND_NUM = 9

// Handler routines for specific RTSP commands:
var allowedCommandNames [MAX_COMMAND_NUM]string = [MAX_COMMAND_NUM]string{"OPTIONS", "DESCRIBE", "SETUP", "TEARDOWN", "PLAY", "PAUSE", "RECORD", "GET_PARAMETER", "SET_PARAMETER"}

type RTSPRequestInfo struct {
	cseq         string
	cmdName      string
	sessionIdStr string
    urlPreSuffix string
    urlSuffix    string
    contentLength uint
}

type TransportHeader struct {
    streamingMode     int
    clientRTPPortNum  int
    clientRTCPPortNum int
    rtpChannelId      int
    rtcpChannelId     int
    destinationTTL    int
    destinationAddr   string
    streamingModeStr  string
}

const (
    RTP_UDP = iota
    RTP_TCP
    RAW_UDP
)

func ParseRTSPRequestString(buf []byte, length int) (*RTSPRequestInfo, bool) {
	reqStr := string(buf)

	var result bool
	reqInfo := new(RTSPRequestInfo)
	reqInfo.cmdName, result = parseCommandName(reqStr)
	if !result {
		return nil, false
	}
	/*
		reqInfo.cseq, result = parseRequestCSeq(reqStr[len(reqInfo.cmdName):])
		if !result {
			return nil, false
		}
	*/
	return reqInfo, result
}

func parseCommandName(reqStr string) (string, bool) {
	var result bool
	var cmdName string
	for _, value := range allowedCommandNames {
		if strings.HasPrefix(string(reqStr), value) {
			cmdName, result = value, true
			break
		}
	}

	return cmdName, result
}

func parseRequestCSeq(reqStr string) (string, bool) {
	cseqIndex := strings.Index(reqStr, "CSeq:")

	ok := false
	index := 0
	for {
		if cseqIndex+1 >= len(reqStr) {
			break
		}

		if reqStr[cseqIndex] == '\r' && reqStr[cseqIndex+1] == '\n' {
			ok = true
			break
		}

		index += 1
		cseqIndex += 1
	}

	var cseq string
	if ok {
		cseq = strings.Trim(reqStr[cseqIndex-index+5:cseqIndex], " ")
	}

	return cseq, true
}

func parseTransportHeader(reqStr string) (*TransportHeader, bool) {
    // Initialize the result parameters to default values:
    header := new(TransportHeader)
    header.streamingMode = RTP_UDP
    header.destinationTTL = 255
    header.clientRTPPortNum  = 0
    header.clientRTCPPortNum = 1
    header.rtpChannelId  = 0xFF
    header.rtcpChannelId = 0xFF
    return header, true
}

// A "Date:" header that can be used in a RTSP (or HTTP) response
func DateHeader() string {
	return fmt.Sprintf("Date: %s\r\n", time.Now())
}
