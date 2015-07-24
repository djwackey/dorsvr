package liveMedia

import (
	"fmt"
    "os"
	. "groupsock"
	"strings"
)

//////// MediaSession ////////
type MediaSession struct {
    cname            string
	controlPath      string
	absStartTime     string
	absEndTime       string
	subSessionNum    int
	subSessionIndex  int
	mediaSubSessions []*MediaSubSession
}

func NewMediaSession(sdpDesc string) *MediaSession {
	mediaSession := new(MediaSession)
	mediaSession.mediaSubSessions = make([]*MediaSubSession, 1024)
	mediaSession.cname, _ = os.Hostname()
	mediaSession.InitWithSDP(sdpDesc)
	return mediaSession
}

func (this *MediaSession) InitWithSDP(sdpDesc string) bool {
    if len(sdpDesc) < 1 {
        return false
    }

    for {
        nextSDPLine, result := this.parseSDPLine(sdpDesc)
        if !result {
            return false
        }

        if sdpDesc[0] == "m" {
            break
        }

        // Check for various special SDP lines that we understand:
        if this.parseSDPLine_s(sdpLine) {
            continue
        }
        if this.parseSDPLine_i(sdpLine) {
            continue
        }
        if this.parseSDPLine_c(sdpLine) {
            continue
        }
        if this.parseSDPAttribute_control(sdpLine) {
            continue
        }
        if this.parseSDPAttribute_range(sdpLine) {
            continue
        }
        if this.parseSDPAttribute_type(sdpLine) {
            continue
        }
        if this.parseSDPAttribute_source_filter(sdpLine) {
            continue
        }
    }

    var num int
    var mediumName, protocolName, payloadFormat string
    for {
	    subsession := NewMediaSubSession()
        if subsession == nil {
            fmt.Println("Unable to create new MediaSubsession")
            return false
        }

        if (num, _ = fmt.Sscanf(sdpLine, "m=%s %d RTP/AVP %d",     mediumName, subsession.clientPortNum, payloadFormat); num == 3 ||
            num, _ = fmt.Sscanf(sdpLine, "m=%s %d/%*d RTP/AVP %d", mediumName, subsession.clientPortNum, payloadFormat); num == 3) && int(payloadFormat) <= 127 {
            protocolName = "RTP"
        } else if (num, _ = fmt.Sscanf(sdpLine, "m=%s %d UDP %d", mediumName, subsession.clientPortNum, payloadFormat); num == 3 ||
                   num, _ = fmt.Sscanf(sdpLine, "m=%s %d udp %d", mediumName, subsession.clientPortNum, payloadFormat); num == 3 ||
                   num, _ = fmt.Sscanf(sdpLine, "m=%s %d RAW/RAW/UDP %d", mediumName, subsession.clientPortNum, payloadFormat); num == 3) && int(payloadFormat) <= 127 {
            // This is a RAW UDP source
            protocolName = "UDP"
        } else {
        }

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
        for {
            sdpLine = nextSDPLine
            if len(sdpLine) < 1 {
                break; // we've reached the end
            }
            if !parseSDPLine(sdpLine, nextSDPLine)
                return false

            if sdpLine[0] == "m" {
                break // we've reached the next subsession
            }

            // Check for various special SDP lines that we understand:
            if subsession->parseSDPLine_c(sdpLine) {
                continue
            }
            if subsession->parseSDPLine_b(sdpLine) {
                continue
            }
            if subsession->parseSDPAttribute_rtpmap(sdpLine) {
                continue
            }
            if subsession->parseSDPAttribute_control(sdpLine) {
                continue
            }
            if subsession->parseSDPAttribute_range(sdpLine) {
                continue
            }
            if subsession->parseSDPAttribute_fmtp(sdpLine)) continue
                                                                                        if subsession->parseSDPAttribute_source_filter(sdpLine)) continue
                                                                                              if subsession->parseSDPAttribute_x_dimensions(sdpLine)) continue;
                                                                                                    if (subsession->parseSDPAttribute_framerate(sdpLine)) continue;

                                                                                                          // (Later, check for malformed lines, and other valid SDP lines#####)
                                                                                                              }

        if len(this.codecName) < 1 {
            subsession.codecName = this.lookupPayloadFormat(subsession.rtpPayloadFormat, subsession.rtpTimestampFrequency, subsession.numChannels)
        }

        // If we don't yet know this subsession's RTP timestamp frequency
        // (because it uses a dynamic payload type and the corresponding
        // SDP "rtpmap" attribute erroneously didn't specify it),
        // then guess it now:
        if subsession.rtpTimestampFrequency == 0 {
            subsession.rtpTimestampFrequency = this.guessRTPTimestampFrequency(subsession.mediumName, subsession.codecName)
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
    var result bool
    for i:=0; i<len(inputLine); i++ {
        if inputLine[i] == "\r" || inputLine[i] == "\n" {
            for {
                if inputLine[i] != "\r" && inputLine[i] != "\n" {
                    break
                }
                i++
            }
            nextLine, result = inputLine[i:], true
            break
        }
    }

    // Then, check that this line is a SDP line of the form <char>=<etc>
    // (However, we also accept blank lines in the input.)
    if inputLine[0] == "\r" || inputLine[0] == "\n" {
        return nextLine, true
    }
    if len(inputLine) < 2 || inputLine[1] != "=" || inputLine[0] < "a" || inputLine[0] > "z" {
        fmt.Println("Invalid SDP line: ", inputLine)
        return nextLine, false
    }

    return nextLine, true
}

func (this *MediaSession) parseCLine() {
}

func (this *MediaSession) parseSDPLine_s(inputLine string) bool {
}

func (this *MediaSession) parseSDPLine_i(inputLine string) bool {
}

func (this *MediaSession) parseSDPLine_c(inputLine string) bool {
}

func (this *MediaSession) parseSDPAttribute_type(sdpLine string) {
}

func (this *MediaSession) parseSDPAttribute_control(sdpLine string) {
}

func (this *MediaSession) parseRangeAttribute() {
}

func (this *MediaSession) parseSDPAttribute_range() {
}

func (this *MediaSession) parseSourceFilterAttribute() {
}

func (this *MediaSession) parseSDPAttribute_source_filter() {
}

func (this *MediaSession) lookupPayloadFormat() {
}

func (this *MediaSession) guessRTPTimestampFrequency() {
}

func (this *MediaSession) initiateByMediaType() {
}


//////// MediaSubSession ////////
type MediaSubSession struct {
	rtpSocket             *GroupSock
	rtcpSocket            *GroupSock
	Sink                  IMediaSink
	rtpSource             *RTPSource
	readSource            IFramedSource
	rtcpInstance          *RTCPInstance
	rtpTimestampFrequency uint
	rtpPayloadFormat      int
	clientPortNum         uint
	protocolName          string
	controlPath           string
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

	protocolIsRTP := strings.EqualFold(this.protocolName, "RTP")
	if protocolIsRTP {
		this.clientPortNum = this.clientPortNum &^ 1
	}

	this.rtpSocket = NewGroupSock(this.clientPortNum)
	if this.rtpSocket == nil {
		fmt.Println("Failed to create RTP socket")
		return false
	}

	if protocolIsRTP {
		// Set our RTCP port to be the RTP Port +1
		rtcpPortNum := this.clientPortNum | 1
		this.rtcpSocket = NewGroupSock(rtcpPortNum)
	}

	//this.rtcpInstance = NewRTCPInstance()
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
