package liveMedia

import (
	"fmt"
	. "groupsock"
	"os"
	"strconv"
	"strings"
)

//////// MediaSession ////////
type MediaSession struct {
	cname                  string
	sessionName            string
	controlPath            string
	absStartTime           string
	absEndTime             string
	mediaSessionType       string
	sessionDescription     string
	connectionEndpointName string
	subSessionNum          int
	subSessionIndex        int
	maxPlayStartTime       float64
	maxPlayEndTime         float64
	mediaSubSessions       []*MediaSubSession
}

func NewMediaSession(sdpDesc string) *MediaSession {
	mediaSession := new(MediaSession)
	mediaSession.mediaSubSessions = make([]*MediaSubSession, 1024)
	mediaSession.cname, _ = os.Hostname()
	mediaSession.InitWithSDP(sdpDesc)
	return mediaSession
}

func (this *MediaSession) InitWithSDP(sdpDesc string) bool {
	sdpLine := sdpDesc
	if len(sdpLine) < 1 {
		return false
	}

	var result bool
	var nextSDPLine string
	for {
		nextSDPLine, result = this.parseSDPLine(sdpLine)
		if !result {
			return false
		}

		if sdpDesc[0] == 'm' {
			break
		}

		fmt.Println(nextSDPLine)

		// Check for various special SDP lines that we understand:
		//if this.parseSDPLine_s(sdpLine) {
		//    continue
		//}
		//if this.parseSDPLine_i(sdpLine) {
		//    continue
		//}
		//if this.parseSDPLine_c(sdpLine) {
		//    continue
		//}
		//if this.parseSDPAttribute_control(sdpLine) {
		//    continue
		//}
		//if this.parseSDPAttribute_range(sdpLine) {
		//    continue
		//}
		//if this.parseSDPAttribute_type(sdpLine) {
		//    continue
		//}
		//if this.parseSDPAttribute_source_filter(sdpLine) {
		//    continue
		//}
	}

	//var num int
	var payloadFormat uint
	var mediumName, protocolName string
	for {
		subsession := NewMediaSubSession()
		if subsession == nil {
			fmt.Println("Unable to create new MediaSubsession")
			return false
		}
		/*
		   if (num, _ = fmt.Sscanf(sdpLine, "m=%s %d RTP/AVP %d", mediumName, subsession.clientPortNum, payloadFormat); num == 3 || num, _ = fmt.Sscanf(sdpLine, "m=%s %d/%*d RTP/AVP %d", mediumName, subsession.clientPortNum, payloadFormat); num == 3) && int(payloadFormat) <= 127 {
		       protocolName = "RTP"
		   } else if (num, _ = fmt.Sscanf(sdpLine, "m=%s %d UDP %d", mediumName, subsession.clientPortNum, payloadFormat); num == 3 ||
		              num, _ = fmt.Sscanf(sdpLine, "m=%s %d udp %d", mediumName, subsession.clientPortNum, payloadFormat); num == 3 ||
		              num, _ = fmt.Sscanf(sdpLine, "m=%s %d RAW/RAW/UDP %d", mediumName, subsession.clientPortNum, payloadFormat); num == 3) && int(payloadFormat) <= 127 {
		       // This is a RAW UDP source
		       protocolName = "UDP"
		   } else {
		   }
		*/
		// Insert this subsession at the end of the list:
		//this.mediaSubSessions = append(this.mediaSubSessions, subsession)
		this.mediaSubSessions[this.subSessionNum] = subsession
		this.subSessionNum++

		subsession.serverPortNum = subsession.clientPortNum
		subsession.savedSDPLines = sdpLine
		subsession.mediumName = mediumName
		subsession.protocolName = protocolName
		subsession.rtpPayloadFormat = payloadFormat

		// Process the following SDP lines, up until the next "m=":
		//        for {
		//            sdpLine = nextSDPLine
		//            if len(sdpLine) < 1 {
		//                break; // we've reached the end
		//            }
		//            if !this.parseSDPLine(sdpLine, nextSDPLine) {
		//                return false
		//            }
		//
		//            if sdpLine[0] == 'm' {
		//                break // we've reached the next subsession
		//            }
		//
		//            // Check for various special SDP lines that we understand:
		//            if subsession.parseSDPLine_c(sdpLine) {
		//                continue
		//            }
		//            if subsession.parseSDPLine_b(sdpLine) {
		//                continue
		//            }
		//            if subsession.parseSDPAttribute_rtpmap(sdpLine) {
		//                continue
		//            }
		//            if subsession.parseSDPAttribute_control(sdpLine) {
		//                continue
		//            }
		//            if subsession.parseSDPAttribute_range(sdpLine) {
		//                continue
		//            }
		//            if subsession.parseSDPAttribute_fmtp(sdpLine) {
		//                continue
		//            }
		//            if subsession.parseSDPAttribute_source_filter(sdpLine) {
		//                continue
		//            }
		//            if subsession.parseSDPAttribute_x_dimensions(sdpLine) {
		//                continue
		//            }
		//            if subsession.parseSDPAttribute_framerate(sdpLine) {
		//                continue
		//            }
		//            // (Later, check for malformed lines, and other valid SDP lines#####)
		//        }
		//
		if len(subsession.codecName) < 1 {
			//subsession.codecName = this.lookupPayloadFormat(subsession.rtpPayloadFormat, subsession.rtpTimestampFrequency, subsession.numChannels)
		}

		// If we don't yet know this subsession's RTP timestamp frequency
		// (because it uses a dynamic payload type and the corresponding
		// SDP "rtpmap" attribute erroneously didn't specify it),
		// then guess it now:
		if subsession.rtpTimestampFrequency == 0 {
			//subsession.rtpTimestampFrequency = this.guessRTPTimestampFrequency(subsession.mediumName, subsession.codecName)
		}
	}
}

func (this *MediaSession) ControlPath() string {
	return this.controlPath
}

func (this *MediaSession) AbsStartTime() string {
	return this.absStartTime
}

func (this *MediaSession) AbsEndTime() string {
	return this.absEndTime
}

func (this *MediaSession) HasSubSessions() bool {
	return len(this.mediaSubSessions) > 0
}

func (this *MediaSession) SubSession() *MediaSubSession {
	this.subSessionIndex++
	return this.mediaSubSessions[this.subSessionIndex-1]
}

func (this *MediaSession) parseSDPLine(inputLine string) (string, bool) {
	// Begin by finding the start of the next line (if any):
	var nextLine string
	for i := 0; i < len(inputLine); i++ {
		if inputLine[i] == '\r' || inputLine[i] == '\n' {
			for {
				if inputLine[i] != '\r' && inputLine[i] != '\n' {
					break
				}
				i++
			}
			nextLine = inputLine[i:]
			break
		}
	}

	// Then, check that this line is a SDP line of the form <char>=<etc>
	// (However, we also accept blank lines in the input.)
	if inputLine[0] == '\r' || inputLine[0] == '\n' {
		return nextLine, true
	}
	if len(inputLine) < 2 || inputLine[1] != '=' || inputLine[0] < 'a' || inputLine[0] > 'z' {
		fmt.Println("Invalid SDP line: ", inputLine)
		return nextLine, false
	}

	return nextLine, true
}

func parseCLine(sdpLine string) string {
	var result string
	fmt.Sscanf(sdpLine, "c=IN IP4 %[^/\r\n]", result)
	return result
}

func (this *MediaSession) parseSDPLine_s(sdpLine string) bool {
	// Check for "s=<session name>" line
	var parseSuccess bool

	var buffer string
	if num, _ := fmt.Sscanf(sdpLine, "s=%[^\r\n]", buffer); num == 1 {
		this.sessionName = buffer
		parseSuccess = true
	}

	return parseSuccess
}

func (this *MediaSession) parseSDPLine_i(sdpLine string) bool {
	// Check for "i=<session description>" line
	var parseSuccess bool

	var buffer string
	if num, _ := fmt.Sscanf(sdpLine, "i=%[^\r\n]", buffer); num == 1 {
		this.sessionDescription = buffer
		parseSuccess = true
	}

	return parseSuccess
}

func (this *MediaSession) parseSDPLine_c(sdpLine string) bool {
	// Check for "c=IN IP4 <connection-endpoint>"
	// or "c=IN IP4 <connection-endpoint>/<ttl+numAddresses>"
	// (Later, do something with <ttl+numAddresses> also #####)
	connectionEndpointName := parseCLine(sdpLine)
	if connectionEndpointName != "" {
		this.connectionEndpointName = connectionEndpointName
		return true
	}

	return false
}

func (this *MediaSession) parseSDPAttribute_type(sdpLine string) bool {
	// Check for a "a=type:broadcast|meeting|moderated|test|H.332|recvonly" line:
	var parseSuccess bool

	var buffer string
	if num, _ := fmt.Sscanf(sdpLine, "a=type: %[^ ]", buffer); num == 1 {
		this.mediaSessionType = buffer
		parseSuccess = true
	}

	return parseSuccess
}

func (this *MediaSession) parseSDPAttribute_control(sdpLine string) bool {
	// Check for a "a=control:<control-path>" line:
	var parseSuccess bool

	var controlPath string
	if num, _ := fmt.Sscanf(sdpLine, "a=control: %s", controlPath); num == 1 {
		parseSuccess = true
		this.controlPath = controlPath
	}

	return parseSuccess
}

func (this *MediaSession) parseRangeAttribute(sdpLine, method string) (string, string, bool) {
	if method == "npt" {
		var startTime, endTime string
		num, _ := fmt.Sscanf(sdpLine, "a=range: npt = %lg - %lg", startTime, endTime)
		return startTime, endTime, (num == 2)
	} else if method == "clock" {
		var as, ae, absStartTime, absEndTime string
		num, _ := fmt.Sscanf(sdpLine, "a=range: clock = %[^-\r\n]-%[^\r\n]", as, ae)
		if num == 2 {
			absStartTime = as
			absEndTime = ae
		} else if num == 1 {
			absStartTime = as
		}

		return absStartTime, absEndTime, (num == 2) || (num == 1)
	}

	return "", "", false
}

func (this *MediaSession) parseSDPAttribute_range(sdpLine string) bool {
	// Check for a "a=range:npt=<startTime>-<endTime>" line:
	// (Later handle other kinds of "a=range" attributes also???#####)
	var parseSuccess bool

	startTime, endTime, ret := this.parseRangeAttribute(sdpLine, "npt")
	if ret {
		parseSuccess = true

		playStartTime, _ := strconv.ParseFloat(startTime, 32)
		playEndTime, _ := strconv.ParseFloat(endTime, 32)

		if playStartTime > this.maxPlayStartTime {
			this.maxPlayStartTime = playStartTime
		}
		if playEndTime > this.maxPlayEndTime {
			this.maxPlayEndTime = playEndTime
		}
	} else if this.absStartTime, this.absEndTime, ret = this.parseRangeAttribute(sdpLine, "clock"); ret {
		parseSuccess = true
	}

	return parseSuccess
}

func (this *MediaSession) parseSourceFilterAttribute(sdpLine string) (string, bool) {
	// Check for a "a=source-filter:incl IN IP4 <something> <source>" line.
	// Note: At present, we don't check that <something> really matches
	// one of our multicast addresses.  We also don't support more than
	// one <source> #####
	var result bool // until we succeed
	var sourceName string
	if num, _ := fmt.Sscanf(sdpLine, "a=source-filter: incl IN IP4 %*s %s", sourceName); num == 1 {
		result = true
	}
	return sourceName, result
}

func (this *MediaSession) parseSDPAttribute_source_filter(sdpLine string) (string, bool) {
	return this.parseSourceFilterAttribute(sdpLine)
}

func (this *MediaSession) lookupPayloadFormat(rtpPayloadType uint) (string, uint, uint) {
	// Look up the codec name and timestamp frequency for known (static)
	// RTP payload formats.
	var temp string
	var freq, nCh uint
	switch rtpPayloadType {
	case 0:
		temp = "PCMU"
		freq = 8000
		nCh = 1
	case 2:
		temp = "G726-32"
		freq = 8000
		nCh = 1
	case 3:
		temp = "GSM"
		freq = 8000
		nCh = 1
	case 4:
		temp = "G723"
		freq = 8000
		nCh = 1
	case 5:
		temp = "DVI4"
		freq = 8000
		nCh = 1
	case 6:
		temp = "DVI4"
		freq = 16000
		nCh = 1
	case 7:
		temp = "LPC"
		freq = 8000
		nCh = 1
	case 8:
		temp = "PCMA"
		freq = 8000
		nCh = 1
	case 9:
		temp = "G722"
		freq = 8000
		nCh = 1
	case 10:
		temp = "L16"
		freq = 44100
		nCh = 2
	case 11:
		temp = "L16"
		freq = 44100
		nCh = 1
	case 12:
		temp = "QCELP"
		freq = 8000
		nCh = 1
	case 14:
		temp = "MPA"
		freq = 90000
		nCh = 1
	// 'number of channels' is actually encoded in the media stream
	case 15:
		temp = "G728"
		freq = 8000
		nCh = 1
	case 16:
		temp = "DVI4"
		freq = 11025
		nCh = 1
	case 17:
		temp = "DVI4"
		freq = 22050
		nCh = 1
	case 18:
		temp = "G729"
		freq = 8000
		nCh = 1
	case 25:
		temp = "CELB"
		freq = 90000
		nCh = 1
	case 26:
		temp = "JPEG"
		freq = 90000
		nCh = 1
	case 28:
		temp = "NV"
		freq = 90000
		nCh = 1
	case 31:
		temp = "H261"
		freq = 90000
		nCh = 1
	case 32:
		temp = "MPV"
		freq = 90000
		nCh = 1
	case 33:
		temp = "MP2T"
		freq = 90000
		nCh = 1
	case 34:
		temp = "H263"
		freq = 90000
		nCh = 1
	}

	return temp, freq, nCh
}

func (this *MediaSession) guessRTPTimestampFrequency(mediumName, codecName string) uint {
	// By default, we assume that audio sessions use a frequency of 8000,
	// video sessions use a frequency of 90000,
	// and text sessions use a frequency of 1000.
	// Begin by checking for known exceptions to this rule
	// (where the frequency is known unambiguously (e.g., not like "DVI4"))
	if strings.EqualFold(codecName, "L16") {
		return 44100
	}
	if strings.EqualFold(codecName, "MPA") ||
		strings.EqualFold(codecName, "MPA-ROBUST") ||
		strings.EqualFold(codecName, "X-MP3-DRAFT-00") {
		return 90000
	}

	// Now, guess default values:
	if strings.EqualFold(mediumName, "video") {
		return 90000
	} else if strings.EqualFold(mediumName, "text") {
		return 1000
	}
	return 8000 // for "audio", and any other medium
}

func (this *MediaSession) initiateByMediaType(mimeType string, useSpecialRTPoffset int) bool {
	return true
}

//////// MediaSubSession ////////
type MediaSubSession struct {
	rtpSocket             *GroupSock
	rtcpSocket            *GroupSock
	Sink                  IMediaSink
	rtpSource             *RTPSource
	readSource            IFramedSource
	rtcpInstance          *RTCPInstance
	parent                *MediaSession
	rtpTimestampFrequency uint
	rtpPayloadFormat      uint
	clientPortNum         uint
	serverPortNum         uint
	numChannels           uint
	bandwidth             uint
	protocolName          string
	controlPath           string
	savedSDPLines         string
	mediumName            string
	codecName             string
}

func NewMediaSubSession() *MediaSubSession {
	subsession := new(MediaSubSession)
	return subsession
}

func (this *MediaSubSession) Initiate() bool {
	if len(this.codecName) <= 0 {
		fmt.Println("Codec is unspecified")
		return false
	}

	tempAddr := ""

	protocolIsRTP := strings.EqualFold(this.protocolName, "RTP")
	if protocolIsRTP {
		this.clientPortNum = this.clientPortNum &^ 1
	}

	this.rtpSocket = NewGroupSock(tempAddr, this.clientPortNum)
	if this.rtpSocket == nil {
		fmt.Println("Failed to create RTP socket")
		return false
	}

	if protocolIsRTP {
		// Set our RTCP port to be the RTP Port +1
		rtcpPortNum := this.clientPortNum | 1
		this.rtcpSocket = NewGroupSock(tempAddr, rtcpPortNum)
	}

	var totSessionBandwidth uint
	if this.bandwidth != 0 {
		totSessionBandwidth = this.bandwidth + this.bandwidth/20
	} else {
		totSessionBandwidth = 500
	}
	this.rtcpInstance = NewRTCPInstance(this.rtcpSocket, totSessionBandwidth, this.parent.cname)
	return true
}

func (this *MediaSubSession) deInitiate() {
}

func (this *MediaSubSession) ControlPath() string {
	return this.controlPath
}

func (this *MediaSubSession) RtcpInstance() *RTCPInstance {
	return this.rtcpInstance
}

func (this *MediaSubSession) createSourceObject() {
	if strings.EqualFold(this.protocolName, "RTP") {
		this.readSource = NewBasicUDPSource()
		this.rtpSource = nil

		if strings.EqualFold(this.codecName, "MP2T") {
			// this sets "durationInMicroseconds" correctly, based on the PCR values
			//this.readSource = NewMPEG2TransportStreamFramer(this.readSource)
		}
	} else {
		switch this.codecName {
		case "H264":
			//this.readSource = NewH264VideoRTPSource(this.rtpSocket, this.rtpPayloadFormat, this.rtpTimestampFrequency)
		}
	}
}
