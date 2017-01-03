package livemedia

import (
	"fmt"
	"strings"
	"time"
)

const maxCommandNum = 9

// Handler routines for specific RTSP commands:
var AllowedCommandNames [maxCommandNum]string = [maxCommandNum]string{
	"OPTIONS",
	"DESCRIBE",
	"SETUP",
	"TEARDOWN",
	"PLAY",
	"PAUSE",
	"RECORD",
	"GET_PARAMETER",
	"SET_PARAMETER",
}

type RTSPRequestInfo struct {
	Cseq          string
	CmdName       string
	SessionIDStr  string
	UrlPreSuffix  string
	UrlSuffix     string
	ContentLength string
}

type HTTPRequestInfo struct {
	CmdName       string
	UrlPreSuffix  string
	UrlSuffix     string
	AcceptStr     string
	SessionCookie string
}

type TransportHeader struct {
	StreamingMode     uint
	ClientRTPPortNum  uint
	ClientRTCPPortNum uint
	RTPChannelID      uint
	RTCPChannelID     uint
	DestinationTTL    uint
	DestinationAddr   string
	StreamingModeStr  string
}

type RangeHeader struct {
	RangeStart   float32
	RangeEnd     float32
	AbsStartTime string
	AbsEndTime   string
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
		reqInfo.CmdName += string(reqStr[i])
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
					reqInfo.UrlSuffix += string(reqStr[k2])
				}
			}
			k2 = i + 1
			if i <= k {
				for ; k2 <= k1-1; k2++ {
					reqInfo.UrlPreSuffix += string(reqStr[k2])
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
				reqInfo.Cseq += string(reqStr[j])
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
				reqInfo.SessionIDStr += string(reqStr[j])
			}
		}
	}

	// Also: Look for "Content-Length:" (optional, case insensitive)
	for j = i; j < reqStrSize-15; j++ {
		if strings.EqualFold("Content-Length:", reqStr[j:j+15]) {
			j += 15
			for ; j < reqStrSize && (reqStr[j] == ' ' || reqStr[j] == '\t'); j++ {
			}
			if num, _ := fmt.Sscanf(reqStr[j:j+15], "%d", &reqInfo.ContentLength); num == 1 {
				break
			}
		}
	}

	return reqInfo, true
}

func ParseHTTPRequestString(reqStr string, reqStrSize int) (*HTTPRequestInfo, bool) {
	reqInfo := new(HTTPRequestInfo)

	i := 0
	for i = 0; i < reqStrSize && reqStr[i] != ' ' && reqStr[i] != '\t'; i++ {
		reqInfo.CmdName += string(reqStr[i])
	}
	if i >= reqStrSize {
		return nil, false // parse failed
	}

	// Look for the string "HTTP/", before the first \r or \n:
	for ; i < reqStrSize && reqStr[i] == '\r' && reqStr[i] == '\n'; i++ {
		if reqStr[i+0] == 'H' &&
			reqStr[i+1] == 'T' &&
			reqStr[i+2] == 'T' &&
			reqStr[i+3] == 'P' &&
			reqStr[i+4] == '/' {
			i += 5
			break
		}
	}

	// Look for various headers that we're interested in:
	reqInfo.SessionCookie, _ = lookForHeader("x-sessioncookie", reqStr[i:], reqStrSize-i)
	reqInfo.AcceptStr, _ = lookForHeader("Accept", reqStr[i:], reqStrSize-i)
	return reqInfo, true
}

func lookForHeader(headerName string, source string, sourceLen int) (string, int) {
	headerNameLen := len(headerName)
	var resultStr string
	var resultMaxSize int
	for i := 0; i < (sourceLen - headerNameLen); i++ {
		if strings.EqualFold(source[i:], headerName) && source[i+headerNameLen] == ':' {
			for i += headerNameLen + 1; i < sourceLen && (source[i] == ' ' || source[i] == '\t'); i++ {
			}
			for j := i; j < sourceLen; j++ {
				if source[j] == '\r' || source[j] == '\n' {
					resultStr = string(source[i:])
					break
				}
			}
		}
	}
	return resultStr, resultMaxSize
}

func ParseTransportHeader(reqStr string) *TransportHeader {
	// Initialize the result parameters to default values:
	header := new(TransportHeader)
	header.StreamingMode = RTP_UDP
	header.DestinationTTL = 255
	header.ClientRTPPortNum = 0
	header.ClientRTCPPortNum = 1
	header.RTPChannelID = 0xFF
	header.RTCPChannelID = 0xFF

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
				header.StreamingMode = RTP_TCP
			} else if strings.EqualFold(field, "RAW/RAW/UDP") ||
				strings.EqualFold(field, "MP2T/H2221/UDP") {
				header.StreamingMode = RAW_UDP
				header.StreamingModeStr = field
			} else if strings.Index(field, "destination=") != -1 {
				header.DestinationAddr = field[12:]
			} else if num, _ = fmt.Sscanf(field, "ttl%d", &ttl); num == 1 {
				header.DestinationTTL = ttl
			} else if num, _ = fmt.Sscanf(field, "client_port=%d-%d", &p1, &p2); num == 2 {
				header.ClientRTPPortNum = p1
				if header.StreamingMode == RAW_UDP {
					header.ClientRTCPPortNum = 0
				} else {
					header.ClientRTCPPortNum = p2
				}
			} else if num, _ = fmt.Sscanf(field, "client_port=%s", &p1); num == 1 {
				header.ClientRTPPortNum = p1
				if header.StreamingMode == RAW_UDP {
					header.ClientRTCPPortNum = 0
				} else {
					header.ClientRTCPPortNum = p1
				}
			} else if num, _ = fmt.Sscanf(field, "interleaved=%d-%d", &rtpCid, &rtcpCid); num == 2 {
				header.RTPChannelID = rtpCid
				header.RTCPChannelID = rtcpCid
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
		rangeHeader.RangeStart = start
		rangeHeader.RangeEnd = end
	} else {
		num, err = fmt.Sscanf(paramStr, "npt = %lf -", &start)
		if err != nil {
			return nil
		}

		if num == 1 {
			rangeHeader.RangeStart = start
		} else {
			if strings.EqualFold(paramStr, "npt = now -") {
				rangeHeader.RangeStart = 0.0
				rangeHeader.RangeEnd = 0.0
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
						rangeHeader.AbsStartTime = as
						rangeHeader.AbsEndTime = ae
					} else if num == 1 {
						rangeHeader.AbsStartTime = as
					}
				} else {
					fmt.Sscanf(paramStr, "smtpe = %n", &numCharsMatched)
				}
			}
		}
	}

	return rangeHeader
}

func ParseRangeHeader(buf string) (*RangeHeader, bool) {
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

func ParsePlayNowHeader(buf string) bool {
	// Find "x-playNow:" header, if present
	var finded bool
	index := strings.Index(buf, "x-playNow:")
	if index != -1 {
		finded = true
	}

	return finded
}

func ParseScaleHeader(buf string) (float32, bool) {
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
			scale, result = sc, true
		}

		break
	}

	return scale, result
}

// A "Date:" header that can be used in a RTSP (or HTTP) response
func DateHeader() string {
	return fmt.Sprintf("Date: %s\r\n", time.Now())
}
