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
	cseq          string
	cmdName       string
	sessionIdStr  string
	urlPreSuffix  string
	urlSuffix     string
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

type RangeHeader struct {
    rangeStart float32
    rangeEnd   float32
    absStartTime string
    absEndTime   string
}

const (
	RTP_UDP = iota
	RTP_TCP
	RAW_UDP
)

func ParseRTSPRequestString(buf []byte) (*RTSPRequestInfo, bool) {
	reqStr := string(buf)

	var result bool
	reqInfo := new(RTSPRequestInfo)

	array := strings.Split(reqStr, "\r\n")
	length := len(array)
	if length <= 1 {
		fmt.Println("Failed to Split \\r\\n")
		return nil, false
	}

	result = parseCommandName(array[0], reqInfo)
	if !result {
		fmt.Println("Failed to Parse Command Name")
		return nil, false
	}

	for i := 1; i < length; i++ {
		Parse(array[i], reqInfo)
	}

	// Parse URL Suffix
	/*
		reqInfo.cseq, result = parseRequestCSeq(reqStr[len(reqInfo.cmdName):])
		if !result {
			return nil, false
		}
	*/

	return reqInfo, result
}

func Parse(reqStr string, reqInfo *RTSPRequestInfo) bool {
	array := strings.Split(reqStr, " ")
	length := len(array)
	if length <= 1 {
		return false
	}

	length = len(array[0])
	switch array[0] {
	case "CSeq:":
		reqInfo.cseq = array[1]
	case "Session:":
		reqInfo.sessionIdStr = array[1]
	}

	return true
}

func ParseHTTPRequestString() (*RTSPRequestInfo, bool) {
	reqInfo := new(RTSPRequestInfo)
	return reqInfo, true
}

func parseCommandName(reqStr string, reqInfo *RTSPRequestInfo) bool {
	array := strings.Split(reqStr, " ")
	//fmt.Println("parseCommandName", reqStr, len(array))
	if len(array) != 3 {
		array = strings.Split(reqStr, "\t")
		if len(array) != 3 {
			return false
		}
	}

	reqInfo.cmdName = array[0]
	switch reqInfo.cmdName {
	case "DESCRIBE":
		s := array[1]
		l := strings.Split(s, "/")
		t := l[len(l)-1]
		l = strings.Split(t, ".")
		if len(l) != 2 {
			return false
		}

		reqInfo.urlPreSuffix = l[0]
		reqInfo.urlSuffix = l[1]
		//fmt.Println("yanfei: ", l[0], l[1])
		//version := array[2]
		//fmt.Println("parseCommandName: " + version)
	case "SETUP":
	}

	return true
}

/*
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
}*/

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

func parseTransportHeader(reqStr string) *TransportHeader {
	// Initialize the result parameters to default values:
	header := new(TransportHeader)
	header.streamingMode = RTP_UDP
	header.destinationTTL = 255
	header.clientRTPPortNum = 0
	header.clientRTCPPortNum = 1
	header.rtpChannelId = 0xFF
	header.rtcpChannelId = 0xFF

    for {
        // First, find "Transport:"
        index := strings.Index(reqStr, "Transport:")
        if index == -1 {
            break
        }

        fields := buf[10:]
        var p1, p2, rtpCid, rtcpCid, ttl int
        var field string
        for {
            num, err := fmt.Sscanf(fields, "%[^;\r\n]", &field)
            if err != nil {
                break
            }

            if num != 1 { break }

            if strings.EqualFold(field, "RTP/AVP/TCP") {
	            header.streamingMode = RTP_TCP
            } else if strings.EqualFold(field, "RAW/RAW/UDP") ||
                      strings.EqualFold(field, "MP2T/H2221/UDP") {
                header.streamingMode = RAW_UDP
                header.streamingModeStr = field
            } else if strings.Index(field, "destination=") != -1 {
                header.destinationAddr = field[12:]
            } else if num, _ = fmt.Sscanf("ttl%d", &ttl); num == 1 {
                header.destinationTTL = ttl
            } else if num, _ = fmt.Sscanf("client_port=%d-%d", &p1, &p2); num == 2 {
                header.clientRTPPortNum = p1
                if header.streamingMode == RAW_UDP {
                    header.clientRTCPPortNum = 0
                } else {
                    header.clientRTCPPortNum = p2
                }
            } else if num, _ = fmt.Sscanf("client_port=%s", &p1); num == 1 {
                header.clientRTPPortNum = p1
                if header.streamingMode == RAW_UDP {
                    header.clientRTCPPortNum = 0
                } else {
                    header.clientRTCPPortNum = p1
                }
            } else if num, _ = fmt.Sscanf("interleaved=%d-%d", &rtpCid, &rtcpCid); num == 2 {
                header.rtpChannelId = rtpCid
                header.rtcpChannelId = rtcpCid
            }

            fields = fields[len(field):]
            i := 0
            for {
                if fields[i] != ";" {
                    break
                }
                i++
            }
            if fields[i] == "\0" || fields[i] == "\r" || fields[i] == "\n" {
                break
            }
        }
        break
    }

	return header
}

func parseRangeParam(paramStr string) *RangeHeader {
	rangeHeader := new(RangeHeader)

    var start, end float32
    numCharsMatched := 0
    num, _ := fmt.Sscanf(paramStr, "npt = %lf - %lf", &start, &end)
    if err != nil {
        return nil
    }

    if num == 2 {
        rangeHeader.rangeStart = start
        rangeHeader.rangeEnd = end
    } else {
        num, err = fmt.Sscanf(paramStr, "npt = %lf -", &start)
        if err != nil {
            return nil
        }

        if num == 1 {
            rangeHeader.rangeStart = start
        } else {
            if strings.EqualFold(paramStr, "npt = now -") {
                rangeHeader.rangeStart = 0.0
                rangeHeader.rangeEnd = 0.0
            } else {
                num, err = fmt.Sscanf(paramStr, "clock = %n", &numCharsMatched)
                if err != nil {
                    return nil
                }

                if numCharsMatched {
                    as, ae := "", ""
                    num, err = fmt.Sscanf(utcTimes, "%[^-]-%s", &as, &ae)
                    if err != nil {
                        return nil
                    }

                    if num == 2 {
                        rangeHeader.absStartTime = as
                        rangeHeader.absEndTime = ae
                    } else if sscanfResult == 1 {
                        rangeHeader.absStartTime = as
                    }
                } else {
                    fmt.Sscanf(paramStr, "smtpe = %n", &numCharsMatched)
                }
            }
        }
    }

	return rangeHeader
}

func parseRangeHeader(buf string, size int) *RangeHeader {
    // First, find "Range:"
    var finded bool
    for i:=0; i<size; i++ {
        if strings.EqualFold(buf, "Range: ") {
            finded = true
            break
        }
    }
    if !finded { return nil }

    return parseRangeParam(buf)
}

func parsePlayNowHeader(buf string) bool {
    // Find "x-playNow:" header, if present
    var finded bool
    index := strings.Index(buf, "x-playNow:")
    if index != -1 {
        finded = true
    }

	return finded
}

func parseScaleHeader(buf string) float32 {
    // Initialize the result parameter to a default value:
    scale = 1.0
    do {
        index := strings.Index(buf, "Scale:")
        if index != -1 {
            break
        }

        fields := buf[index:]
        i := 0
        for {
            if fields[i] != " " {
                break
            }
            i++
        }
        var sc float32
        if num, _ := fmt.Sscanf(fields, "%f", &sc); num == 1  {
            scale = sc
        }

        break
    }

	return scale
}

// A "Date:" header that can be used in a RTSP (or HTTP) response
func DateHeader() string {
	return fmt.Sprintf("Date: %s\r\n", time.Now())
}
