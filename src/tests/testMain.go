package main

import (
	"fmt"
	"strings"
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

	SETUP_COMMAND = "SETUP rtsp://192.168.6.232:8554/track1 RTSP/1.0\r\n" +
		"CSeq: 4\r\n" +
		"User-Agent: LibVLC/2.1.5 (LIVE555 Streaming Media v2014.05.27)\r\n" +
		"Transport: RTP/AVP;unicast;client_port=52294-52295\r\n\r\n"

	PLAY_COMMAND = "PLAY rtsp://192.168.6.232:8554/ RTSP/1.0\r\n" +
		"CSeq: 5\r\n" +
		"User-Agent: LibVLC/2.1.5 (LIVE555 Streaming Media v2014.05.27)\r\n" +
		"Session: 66711EC7\r\n" +
		"Range: npt=0.000-\r\n\r\n"

	TEARDOWN_COMMAND = ""
)

//for i = 0; i < reqStrSize; i++ {
//	c := reqStr[i]
//	if c == ' ' || c == '\t' {
//		parseSucceeded = true
//		break
//	}
//	cmdName[i] = reqStr[i]
//}

//if !parseSucceeded {
//	return false
//}

func parse(reqStr string, reqStrSize int) bool {
	cmdName := make([]byte, 10)

	// Read everything up to the first space as the command name:
	i := 0
	for i = 0; i < reqStrSize && reqStr[i] != ' ' && reqStr[i] != '\t'; i++ {
		cmdName[i] = reqStr[i]
	}
	if i >= reqStrSize {
		return false // parse failed
	}

	// skip over any additional white space
	j := i + 1
	for ; j < reqStrSize && reqStr[j] == ' ' && reqStr[j] == '\t'; j++ {
	}
	for ; j < reqStrSize-8; j++ {
		if (reqStr[j+0] == 'r' || reqStr[j+0] == 'R') &&
		   (reqStr[j+1] == 't' || reqStr[j+1] == 'T') &&
		   (reqStr[j+2] == 's' || reqStr[j+2] == 'S') &&
		   (reqStr[j+3] == 'p' || reqStr[j+3] == 'P') &&
		    reqStr[j+4] == ':' && reqStr[j+5] == '/' {
			j += 6
			if reqStr[j] == '/' {
				j++
				for ; j < reqStrSize && reqStr[j] != '/' && reqStr[j] != ' '; j++ {
				}
			} else {
				j--
			}
			i = j
			break
		}
	}

	fmt.Println("yanfei", i)
	// Look for the URL suffix
	urlSuffixMaxSize := 10
	urlSuffix := make([]byte, urlSuffixMaxSize)
	urlPreSuffix := make([]byte, urlSuffixMaxSize)
	for k := i + 1; k < reqStrSize-5; k++ {
		if reqStr[k+0] == 'R' &&
			reqStr[k+1] == 'T' &&
			reqStr[k+2] == 'S' &&
			reqStr[k+3] == 'P' &&
			reqStr[k+4] == '/' {
			for k--; k >= i && reqStr[k] == ' '; k-- {
			}
			fmt.Println(k)
			k1 := k
			for ; k1 > i && reqStr[k1] != '/'; k1-- {
			}

			n := 0
			k2 := k1 + 1
			if i <= k {
				for ; k2 <= k; k2++ {
					urlSuffix[n] = reqStr[k2]
					n++
				}
			}
			fmt.Println(i, k, string(reqStr[k2:k2+n]))
			n = 0
			k2 = i + 1
			if i <= k {
				for ; k2 <= k1-1; k2++ {
					urlPreSuffix[n] = reqStr[k2]
					n++
				}
			}

			i = k + 7
			break
		}
	}

	// Look for "CSeq:"
	CSeqMaxSize := 3
	CSeq := make([]byte, CSeqMaxSize)
	//fmt.Println(i)
	for j = i; j < reqStrSize-5; j++ {
		if strings.EqualFold("CSeq:", reqStr[j:j+5]) {
			j += 5
			for ; j < reqStrSize && (reqStr[j] == ' ' || reqStr[j] == '\t'); j++ {
			}
			for n := 0; n < CSeqMaxSize && reqStr[j] != '\r' && reqStr[j] != '\n'; n++ {
				CSeq[n] = reqStr[j]
				if n >= CSeqMaxSize {
					break
				}
				j++
			}
		}
	}

	// Look for "Session:"
	sessionIdMaxSize := 20
	sessionId := make([]byte, sessionIdMaxSize)
	for j = i; j < reqStrSize-8; j++ {
		if strings.EqualFold("Session:", reqStr[j:j+8]) {
			j += 8
			for ; j < reqStrSize && (reqStr[j] == ' ' || reqStr[j] == '\t'); j++ {
			}
			for n := 0; n < sessionIdMaxSize && reqStr[j] != '\r' && reqStr[j] != '\n'; n++ {
				sessionId[n] = reqStr[j]
				if n >= sessionIdMaxSize {
					break
				}
				j++
			}
		}
	}

	// Also: Look for "Content-Length:" (optional, case insensitive)
	var contentLengthStr string
	for j = i; j < reqStrSize-15; j++ {
		if strings.EqualFold("Content-Length:", reqStr[j:j+15]) {
			j += 15
			for ; j < reqStrSize && (reqStr[j] == ' ' || reqStr[j] == '\t'); j++ {
			}
			if num, _ := fmt.Sscanf(reqStr[j:j+15], "%d", contentLengthStr); num == 1 {
				break
			}
		}
	}

	fmt.Println(string(urlPreSuffix))
	fmt.Println(string(urlSuffix))
	fmt.Println(string(sessionId))
	fmt.Println(string(cmdName))
	fmt.Println(string(CSeq))
	return true
}

func main() {
	fmt.Println(DESCRIPTION_COMMAND)
	//parse(DESCRIPTION_COMMAND, len(DESCRIPTION_COMMAND))
	parse(DESCRIPTION_COMMAND, len(DESCRIPTION_COMMAND))
}
