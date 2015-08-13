package liveMedia

import (
	"fmt"
	//"strconv"
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
	contentLength string
}

type TransportHeader struct {
	streamingMode     uint
	clientRTPPortNum  uint
	clientRTCPPortNum uint
	rtpChannelId      uint
	rtcpChannelId     uint
	destinationTTL    uint
	destinationAddr   string
	streamingModeStr  string
}

type RangeHeader struct {
	rangeStart   float32
	rangeEnd     float32
	absStartTime string
	absEndTime   string
}

const (
	RTP_UDP = iota
	RTP_TCP
	RAW_UDP
)

func ParseRTSPRequestString(reqStr string, reqStrSize int) (*RTSPRequestInfo, bool) {
	reqInfo := new(RTSPRequestInfo)

	// Read everything up to the first space as the command name:
	i := 0
	for i = 0; i < reqStrSize && reqStr[i] != ' ' && reqStr[i] != '\t'; i++ {
		reqInfo.cmdName += string(reqStr[i])
	}
	if i >= reqStrSize {
		return nil, false // parse failed
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

	// Look for the URL suffix
	for k := i + 1; k < reqStrSize-5; k++ {
		if reqStr[k+0] == 'R' &&
			reqStr[k+1] == 'T' &&
			reqStr[k+2] == 'S' &&
			reqStr[k+3] == 'P' &&
			reqStr[k+4] == '/' {
			for k--; k >= i && reqStr[k] == ' '; k-- {
			}
			k1 := k
			for ; k1 > i && reqStr[k1] != '/'; k1-- {
			}

			k2 := k1 + 1
			if i <= k {
				for ; k2 <= k; k2++ {
					reqInfo.urlSuffix += string(reqStr[k2])
					//n++
				}
			}
			k2 = i + 1
			if i <= k {
				for ; k2 <= k1-1; k2++ {
					reqInfo.urlPreSuffix += string(reqStr[k2])
					//n++
				}
			}

			i = k + 7
			break
		}
	}

	// Look for "CSeq:"
	for j = i; j < reqStrSize-5; j++ {
		if strings.EqualFold("CSeq:", reqStr[j:j+5]) {
			j += 5
			for ; j < reqStrSize && (reqStr[j] == ' ' || reqStr[j] == '\t'); j++ {
			}
			for ; reqStr[j] != '\r' && reqStr[j] != '\n'; j++ {
				reqInfo.cseq += string(reqStr[j])
			}
		}
	}

	// Look for "Session:"
	for j = i; j < reqStrSize-8; j++ {
		if strings.EqualFold("Session:", reqStr[j:j+8]) {
			j += 8
			for ; j < reqStrSize && (reqStr[j] == ' ' || reqStr[j] == '\t'); j++ {
			}
			for ; reqStr[j] != '\r' && reqStr[j] != '\n'; j++ {
				reqInfo.sessionIdStr += string(reqStr[j])
			}
		}
	}

	// Also: Look for "Content-Length:" (optional, case insensitive)
	for j = i; j < reqStrSize-15; j++ {
		if strings.EqualFold("Content-Length:", reqStr[j:j+15]) {
			j += 15
			for ; j < reqStrSize && (reqStr[j] == ' ' || reqStr[j] == '\t'); j++ {
			}
			if num, _ := fmt.Sscanf(reqStr[j:j+15], "%d", reqInfo.contentLength); num == 1 {
				break
			}
		}
	}

	return reqInfo, true
}

func ParseHTTPRequestString() (*RTSPRequestInfo, bool) {
	reqInfo := new(RTSPRequestInfo)
	return reqInfo, true
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

		var num int
		var p1, p2, rtpCid, rtcpCid, ttl uint

		tranStr := reqStr[index+10:]
		fields := strings.Split(tranStr, ";")

		for _, field := range fields {
			field = strings.TrimSpace(field)

			if strings.EqualFold(field, "RTP/AVP/TCP") {
				header.streamingMode = RTP_TCP
			} else if strings.EqualFold(field, "RAW/RAW/UDP") ||
				strings.EqualFold(field, "MP2T/H2221/UDP") {
				header.streamingMode = RAW_UDP
				header.streamingModeStr = field
			} else if strings.Index(field, "destination=") != -1 {
				header.destinationAddr = field[12:]
			} else if num, _ = fmt.Sscanf(field, "ttl%d", &ttl); num == 1 {
				header.destinationTTL = ttl
			} else if num, _ = fmt.Sscanf(field, "client_port=%d-%d", &p1, &p2); num == 2 {
				header.clientRTPPortNum = p1
				if header.streamingMode == RAW_UDP {
					header.clientRTCPPortNum = 0
				} else {
					header.clientRTCPPortNum = p2
				}
			} else if num, _ = fmt.Sscanf(field, "client_port=%s", &p1); num == 1 {
				header.clientRTPPortNum = p1
				if header.streamingMode == RAW_UDP {
					header.clientRTCPPortNum = 0
				} else {
					header.clientRTCPPortNum = p1
				}
			} else if num, _ = fmt.Sscanf(field, "interleaved=%d-%d", &rtpCid, &rtcpCid); num == 2 {
				header.rtpChannelId = rtpCid
				header.rtcpChannelId = rtcpCid
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
	num, err := fmt.Sscanf(paramStr, "npt = %lf - %lf", &start, &end)
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

				if numCharsMatched > 0 {
					as, ae := "", ""
					utcTimes := string(paramStr[numCharsMatched:])
					num, err = fmt.Sscanf(utcTimes, "%[^-]-%s", &as, &ae)
					if err != nil {
						return nil
					}

					if num == 2 {
						rangeHeader.absStartTime = as
						rangeHeader.absEndTime = ae
					} else if num == 1 {
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

func parseRangeHeader(buf string) (*RangeHeader, bool) {
	var rangeParam *RangeHeader
	var result bool

	for {
		// First, find "Range:"
		var finded bool
		for i := 0; i < len(buf); i++ {
			if strings.EqualFold(buf, "Range: ") {
				finded = true
				break
			}
		}
		if !finded {
			break
		}

		rangeParam = parseRangeParam(buf)
		if rangeParam == nil {
			break
		}
		result = true
		break
	}

	return rangeParam, result
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

func parseScaleHeader(buf string) (float32, bool) {
	// Initialize the result parameter to a default value:
	var scale float32 = 1.0
	var result bool
	for {
		index := strings.Index(buf, "Scale:")
		if index == -1 {
			break
		}

		fmt.Println("parseScaleHeader", buf, index)

		fields := buf[index:]
		i := 0
		for {
			if string(fields[i]) != " " {
				break
			}
			i++
		}
		var sc float32
		if num, _ := fmt.Sscanf(fields, "%f", &sc); num == 1 {
			//f, _ := strconv.ParseFloat(sc, 32)
			//scale = float32(f)
			scale = sc
			result = true
		}

		break
	}

	return scale, result
}

// A "Date:" header that can be used in a RTSP (or HTTP) response
func DateHeader() string {
	return fmt.Sprintf("Date: %s\r\n", time.Now())
}
