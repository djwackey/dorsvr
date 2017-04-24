package livemedia

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	gs "github.com/djwackey/dorsvr/groupsock"
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
	subsessionNum          int
	subsessionIndex        int
	maxPlayStartTime       float64
	maxPlayEndTime         float64
	scale                  float32
	mediaSubsessions       []*MediaSubsession
}

func NewMediaSession(sdpDesc string) *MediaSession {
	cname, _ := os.Hostname()
	s := &MediaSession{
		cname:            cname,
		scale:            1.0,
		mediaSubsessions: make([]*MediaSubsession, 1024),
	}

	if !s.initWithSDP(sdpDesc) {
		return nil
	}
	return s
}

func (s *MediaSession) initWithSDP(sdpLine string) bool {
	if sdpLine == "" {
		return false
	}

	// Begin by processing All SDP lines until we see the first "m="
	var result bool
	var nextSDPLine, thisSDPLine string
	for {
		nextSDPLine, thisSDPLine, result = s.parseSDPLine(sdpLine)
		if !result {
			return false
		}

		sdpLine = nextSDPLine

		if thisSDPLine[0] == 'm' {
			break
		}

		// there is no m= lines at all
		if sdpLine == "" {
			break
		}

		// Check for various special SDP lines that we understand:
		if s.parseSDPLineS(thisSDPLine) {
			continue
		}
		if s.parseSDPLineI(thisSDPLine) {
			continue
		}
		if s.parseSDPLineC(thisSDPLine) {
			continue
		}
		if s.parseSDPAttributeControl(thisSDPLine) {
			continue
		}
		if s.parseSDPAttributeRange(thisSDPLine) {
			continue
		}
		if s.parseSDPAttributeType(thisSDPLine) {
			continue
		}
		if s.parseSDPAttributeSourceFilter(thisSDPLine) {
			continue
		}
	}

	var payloadFormat uint32
	var n1, n2, n3, n4, n5 int
	var mediumName, protocolName string
	for {
		subsession := NewMediaSubsession(s)
		if subsession == nil {
			fmt.Println("Unable to create new MediaSubsession")
			return false
		}

		n1, _ = fmt.Sscanf(thisSDPLine, "m=%s %d RTP/AVP %d", &mediumName,
			&subsession.clientPortNum, &payloadFormat)
		n2, _ = fmt.Sscanf(thisSDPLine, "m=%s %d/%*d RTP/AVP %d", &mediumName,
			&subsession.clientPortNum, &payloadFormat)
		n3, _ = fmt.Sscanf(thisSDPLine, "m=%s %d UDP %d", &mediumName,
			&subsession.clientPortNum, &payloadFormat)
		n4, _ = fmt.Sscanf(thisSDPLine, "m=%s %d udp %d", &mediumName,
			&subsession.clientPortNum, &payloadFormat)
		n5, _ = fmt.Sscanf(thisSDPLine, "m=%s %d RAW/RAW/UDP %d", &mediumName,
			&subsession.clientPortNum, &payloadFormat)

		if (n1 == 3 || n2 == 3) && payloadFormat <= 127 {
			protocolName = "RTP"
		} else if (n3 == 3 || n4 == 3 || n5 == 3) && payloadFormat <= 127 {
			// This is a RAW UDP source
			protocolName = "UDP"
		} else {
		}

		// Insert this subsession at the end of the list:
		//s.mediaSubSessions = append(s.mediaSubSessions, subsession)
		s.mediaSubsessions[s.subsessionNum] = subsession
		s.subsessionNum++

		subsession.serverPortNum = subsession.clientPortNum
		subsession.savedSDPLines = sdpLine
		subsession.mediumName = mediumName
		subsession.protocolName = protocolName
		subsession.rtpPayloadFormat = payloadFormat

		// Process the following SDP lines, up until the next "m=":
		for {
			sdpLine = nextSDPLine
			if sdpLine == "" {
				//fmt.Println("we've reached the end")
				break // we've reached the end
			}

			nextSDPLine, thisSDPLine, result = s.parseSDPLine(sdpLine)
			if !result {
				return false
			}

			if thisSDPLine[0] == 'm' {
				break // we've reached the next subsession
			}

			// Check for various special SDP lines that we understand:
			if subsession.parseSDPLineC(thisSDPLine) {
				continue
			}
			if subsession.parseSDPLineB(thisSDPLine) {
				continue
			}
			if subsession.parseSDPAttributeRtpmap(thisSDPLine) {
				continue
			}
			if subsession.parseSDPAttributeControl(thisSDPLine) {
				continue
			}
			if subsession.parseSDPAttributeRange(thisSDPLine) {
				continue
			}
			if subsession.parseSDPAttributeFmtp(thisSDPLine) {
				continue
			}
			if subsession.parseSDPAttributeSourceFilter(thisSDPLine) {
				continue
			}
			if subsession.parseSDPAttributeXDimensions(thisSDPLine) {
				continue
			}
			if subsession.parseSDPAttributeFrameRate(thisSDPLine) {
				continue
			}
			// (Later, check for malformed lines, and other valid SDP lines#####)
		}

		if subsession.codecName == "" {
			subsession.codecName,
				subsession.rtpTimestampFrequency,
				subsession.numChannels = s.lookupPayloadFormat(subsession.rtpPayloadFormat)
		}

		// If we don't yet know this subsession's RTP timestamp frequency
		// (because it uses a dynamic payload type and the corresponding
		// SDP "rtpmap" attribute erroneously didn't specify it),
		// then guess it now:
		if subsession.rtpTimestampFrequency == 0 {
			subsession.rtpTimestampFrequency = s.guessRTPTimestampFrequency(subsession.mediumName,
				subsession.codecName)
		}
		break
	}
	return true
}

func (s *MediaSession) Scale() float32 {
	return s.scale
}

func (s *MediaSession) ControlPath() string {
	return s.controlPath
}

func (s *MediaSession) AbsStartTime() string {
	return s.absStartTime
}

func (s *MediaSession) AbsEndTime() string {
	return s.absEndTime
}

func (session *MediaSession) HasSubsessions() bool {
	return len(session.mediaSubsessions) > 0
}

func (s *MediaSession) Subsession() *MediaSubsession {
	s.subsessionIndex++
	return s.mediaSubsessions[s.subsessionIndex-1]
}

func (s *MediaSession) parseSDPLine(inputLine string) (nextLine, thisLine string, result bool) {
	inputLen := len(inputLine)

	// Begin by finding the start of the next line (if any):
	for i := 0; i < inputLen; i++ {
		if inputLine[i] == '\r' || inputLine[i] == '\n' {
			for i += 1; i < inputLen && (inputLine[i] == '\r' || inputLine[i] == '\n'); i++ {
			}
			nextLine = inputLine[i:]
			thisLine = inputLine[:i-2]
			break
		}
	}

	if len(thisLine) < 2 || thisLine[1] != '=' || thisLine[0] < 'a' || thisLine[0] > 'z' {
		fmt.Println("Invalid SDP line:", thisLine, nextLine)
	} else {
		result = true
	}
	return
}

func parseCLine(sdpLine string) (result string) {
	fmt.Sscanf(sdpLine, "c=IN IP4 %s", &result)
	return
}

// Check for "s=<session name>" line
func (s *MediaSession) parseSDPLineS(sdpLine string) bool {
	var parseSuccess bool

	if sdpLine[0] == 's' && sdpLine[1] == '=' {
		s.sessionName = sdpLine[2:]
		parseSuccess = true
	}

	return parseSuccess
}

// Check for "i=<session description>" line
func (s *MediaSession) parseSDPLineI(sdpLine string) bool {
	var parseSuccess bool

	if sdpLine[0] == 'i' && sdpLine[1] == '=' {
		s.sessionDescription = sdpLine[2:]
		parseSuccess = true
	}

	return parseSuccess
}

// Check for "c=IN IP4 <connection-endpoint>"
// or "c=IN IP4 <connection-endpoint>/<ttl+numAddresses>"
// (Later, do something with <ttl+numAddresses> also #####)
func (s *MediaSession) parseSDPLineC(sdpLine string) bool {
	connectionEndpointName := parseCLine(sdpLine)
	if connectionEndpointName != "" {
		s.connectionEndpointName = connectionEndpointName
		return true
	}

	return false
}

// Check for a "a=type:broadcast|meeting|moderated|test|H.332|recvonly" line:
func (s *MediaSession) parseSDPAttributeType(sdpLine string) bool {
	var parseSuccess bool

	var buffer string
	if n, _ := fmt.Sscanf(sdpLine, "a=type: %[^ ]", &buffer); n == 1 {
		s.mediaSessionType = buffer
		parseSuccess = true
	}

	return parseSuccess
}

// Check for a "a=control:<control-path>" line:
func (s *MediaSession) parseSDPAttributeControl(sdpLine string) bool {
	var parseSuccess bool

	if ok := strings.HasPrefix(sdpLine, "a=control:"); ok {
		s.controlPath = sdpLine[10:]
		parseSuccess = true
	}

	return parseSuccess
}

func parseRangeAttribute(sdpLine, method string) (string, string, bool) {
	var n int
	if method == "npt" {
		var startTime, endTime string
		n, _ = fmt.Sscanf(sdpLine, "a=range: npt = %f - %f", &startTime, &endTime)
		return startTime, endTime, (n == 2)
	} else if method == "clock" {
		var as, ae, absStartTime, absEndTime string
		n, _ = fmt.Sscanf(sdpLine, "a=range: clock = %[^-\r\n]-%[^\r\n]", &as, &ae)
		if n == 2 {
			absStartTime = as
			absEndTime = ae
		} else if n == 1 {
			absStartTime = as
		}

		return absStartTime, absEndTime, (n == 2) || (n == 1)
	}

	return "", "", false
}

// Check for a "a=range:npt=<startTime>-<endTime>" line:
// (Later handle other kinds of "a=range" attributes also???#####)
func (s *MediaSession) parseSDPAttributeRange(sdpLine string) bool {
	var parseSuccess bool

	startTime, endTime, ok := parseRangeAttribute(sdpLine, "npt")
	if ok {
		parseSuccess = true

		playStartTime, _ := strconv.ParseFloat(startTime, 32)
		playEndTime, _ := strconv.ParseFloat(endTime, 32)

		if playStartTime > s.maxPlayStartTime {
			s.maxPlayStartTime = playStartTime
		}
		if playEndTime > s.maxPlayEndTime {
			s.maxPlayEndTime = playEndTime
		}
	} else if s.absStartTime, s.absEndTime, ok = parseRangeAttribute(sdpLine, "clock"); ok {
		parseSuccess = true
	}

	return parseSuccess
}

// Check for a "a=source-filter:incl IN IP4 <something> <source>" line.
// Note: At present, we don't check that <something> really matches
// one of our multicast addresses.  We also don't support more than
// one <source> #####
func parseSourceFilterAttribute(sdpLine string) bool {
	var sourceName string
	n, _ := fmt.Sscanf(sdpLine, "a=source-filter: incl IN IP4 %*s %s", &sourceName)
	return (n == 1)
}

func (s *MediaSession) parseSDPAttributeSourceFilter(sdpLine string) bool {
	return parseSourceFilterAttribute(sdpLine)
}

// Look up the codec name and timestamp frequency for known (static)
// RTP payload formats.
func (s *MediaSession) lookupPayloadFormat(rtpPayloadType uint32) (codecName string, freq, ch uint32) {
	switch rtpPayloadType {
	case 0:
		codecName, freq, ch = "PCMU", 8000, 1
	case 2:
		codecName, freq, ch = "G726-32", 8000, 1
	case 3:
		codecName, freq, ch = "GSM", 8000, 1
	case 4:
		codecName, freq, ch = "G723", 8000, 1
	case 5:
		codecName, freq, ch = "DVI4", 8000, 1
	case 6:
		codecName, freq, ch = "DVI4", 16000, 1
	case 7:
		codecName, freq, ch = "LPC", 8000, 1
	case 8:
		codecName, freq, ch = "PCMA", 8000, 1
	case 9:
		codecName, freq, ch = "G722", 8000, 1
	case 10:
		codecName, freq, ch = "L16", 44100, 2
	case 11:
		codecName, freq, ch = "L16", 44100, 1
	case 12:
		codecName, freq, ch = "QCELP", 8000, 1
	case 14:
		codecName, freq, ch = "MPA", 90000, 1
	// 'number of channels' is actually encoded in the media stream
	case 15:
		codecName, freq, ch = "G728", 8000, 1
	case 16:
		codecName, freq, ch = "DVI4", 11025, 1
	case 17:
		codecName, freq, ch = "DVI4", 22050, 1
	case 18:
		codecName, freq, ch = "G729", 8000, 1
	case 25:
		codecName, freq, ch = "CELB", 90000, 1
	case 26:
		codecName, freq, ch = "JPEG", 90000, 1
	case 28:
		codecName, freq, ch = "NV", 90000, 1
	case 31:
		codecName, freq, ch = "H261", 90000, 1
	case 32:
		codecName, freq, ch = "MPV", 90000, 1
	case 33:
		codecName, freq, ch = "MP2T", 90000, 1
	case 34:
		codecName, freq, ch = "H263", 90000, 1
	}
	return
}

// By default, we assume that audio sessions use a frequency of 8000,
// video sessions use a frequency of 90000,
// and text sessions use a frequency of 1000.
// Begin by checking for known exceptions to s rule
// (where the frequency is known unambiguously (e.g., not like "DVI4"))
func (s *MediaSession) guessRTPTimestampFrequency(mediumName, codecName string) uint32 {
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

func (s *MediaSession) initiateByMediaType(mimeType string, useSpecialRTPoffset int) bool {
	return true
}

//////// MediaSubsession ////////
type MediaSubsession struct {
	RTPSource              *RTPSource
	rtpSocket              *gs.GroupSock
	rtcpSocket             *gs.GroupSock
	Sink                   IMediaSink
	readSource             IFramedSource
	rtcpInstance           *RTCPInstance
	parent                 *MediaSession
	MiscPtr                interface{}
	numChannels            uint32
	rtpChannelID           uint
	rtcpChannelID          uint
	rtpPayloadFormat       uint32
	rtpTimestampFrequency  uint32
	clientPortNum          uint
	serverPortNum          uint
	bandWidth              uint
	videoWidth             uint
	videoHeight            uint
	sessionID              string
	protocolName           string
	controlPath            string
	savedSDPLines          string
	mediumName             string
	codecName              string
	absStartTime           string
	absEndTime             string
	connectionEndpointName string
	playStartTime          float64
	playEndTime            float64
	videoFPS               float32
	scale                  float32
}

func NewMediaSubsession(parent *MediaSession) *MediaSubsession {
	if parent == nil {
		return nil
	}

	return &MediaSubsession{
		scale:  1.0,
		parent: parent,
	}
}

func (subsession *MediaSubsession) ParentSession() *MediaSession {
	return subsession.parent
}

func (s *MediaSubsession) Initiate() bool {
	// has already been initiated
	if s.readSource != nil {
		return true
	}

	if len(s.codecName) <= 0 {
		fmt.Println("Codec is unspecified")
		return false
	}

	tempAddr := s.ConnectionEndpointName()

	var success bool
	for {
		// create new socket
		s.rtpSocket = gs.NewGroupSock(tempAddr, 0)
		if s.rtpSocket == nil {
			fmt.Println("Unable to create RTP socket")
			break
		}

		clientPortNum := s.rtpSocket.GetSourcePort()
		if clientPortNum == 0 {
			fmt.Println("Failed to get RTP port number")
			break
		}

		if clientPortNum&1 != 0 {
			s.rtpSocket.Close()
			continue
		}

		s.clientPortNum = clientPortNum

		rtcpPortNum := clientPortNum | 1
		s.rtcpSocket = gs.NewGroupSock(tempAddr, rtcpPortNum)
		if s.rtcpSocket == nil {
			fmt.Println("Unable to create RTCP socket")
			break
		}

		success = true
		break
	}

	if !success {
		if s.rtpSocket != nil {
			s.rtpSocket.Close()
		}
		if s.rtcpSocket != nil {
			s.rtcpSocket.Close()
		}
		return false
	}

	if !s.createSourceObject() {
		return false
	}

	if s.readSource == nil {
		fmt.Println("Failed to create read source.")
		return false
	}

	var totSessionBandwidth uint
	if s.bandWidth != 0 {
		totSessionBandwidth = s.bandWidth + s.bandWidth/20
	} else {
		totSessionBandwidth = 500
	}

	s.rtcpInstance = newRTCPInstance(s.rtcpSocket, totSessionBandwidth, s.parent.cname, nil, s.RTPSource)
	return true
}

func (s *MediaSubsession) Scale() float32 {
	return s.scale
}

func (s *MediaSubsession) SetRTPChannelID(rtpChannelID uint) {
	s.rtpChannelID = rtpChannelID
}

func (s *MediaSubsession) SetRTCPChannelID(rtcpChannelID uint) {
	s.rtcpChannelID = rtcpChannelID
}

func (s *MediaSubsession) SetServerPortNum(serverPortNum uint) {
	s.serverPortNum = serverPortNum
}

func (s *MediaSubsession) SetConnectionEndpointName(connectionEndpointName string) {
	s.connectionEndpointName = connectionEndpointName
}

func (s *MediaSubsession) SetSessionID(sessionID string) {
	s.sessionID = sessionID
}

func (s *MediaSubsession) SessionID() string {
	return s.sessionID
}

func (subsession *MediaSubsession) deInitiate() {
}

func (s *MediaSubsession) AbsStartTime() string {
	if s.absStartTime != "" {
		return s.absStartTime
	}

	return s.parent.AbsStartTime()
}

func (s *MediaSubsession) AbsEndTime() string {
	if s.absEndTime != "" {
		return s.absEndTime
	}

	return s.parent.AbsEndTime()
}

func (s *MediaSubsession) CodecName() string {
	return s.codecName
}

func (s *MediaSubsession) MediumName() string {
	return s.mediumName
}

func (s *MediaSubsession) ClientPortNum() uint {
	return s.clientPortNum
}

func (s *MediaSubsession) ProtocolName() string {
	return s.protocolName
}

func (s *MediaSubsession) ControlPath() string {
	return s.controlPath
}

func (s *MediaSubsession) ReadSource() IFramedSource {
	return s.readSource
}

func (s *MediaSubsession) RtcpInstance() *RTCPInstance {
	return s.rtcpInstance
}

func (s *MediaSubsession) SetDestinations(destAddress string) {
}

func (s *MediaSubsession) ConnectionEndpointName() (name string) {
	name = s.connectionEndpointName
	name, _ = gs.OurIPAddress()
	return
}

func (s *MediaSubsession) createSourceObject() bool {
	if strings.EqualFold(s.protocolName, "UDP") {
		s.readSource = newBasicUDPSource(s.rtpSocket)
		s.RTPSource = nil

		// MPEG-2 Transport Stream
		if strings.EqualFold(s.codecName, "MP2T") {
			// s sets "durationInMicroseconds" correctly, based on the PCR values
			//s.readSource = NewMPEG2TransportStreamFramer(s.readSource)
		}
	} else {
		switch s.codecName {
		case "H264":
			s.readSource = newH264VideoRTPSource(s.rtpSocket,
				s.rtpPayloadFormat, s.rtpTimestampFrequency)
		}
	}
	return true
}

func (s *MediaSubsession) parseSDPLineB(sdpLine string) bool {
	n, _ := fmt.Sscanf(sdpLine, "b=AS:%d", &s.bandWidth)
	return (n == 1)
}

// Check for "c=IN IP4 <connection-endpoint>"
// or "c=IN IP4 <connection-endpoint>/<ttl+numAddresses>"
// (Later, do something with <ttl+numAddresses> also #####)
func (s *MediaSubsession) parseSDPLineC(sdpLine string) bool {
	connectionEndpointName := parseCLine(sdpLine)
	if connectionEndpointName != "" {
		s.connectionEndpointName = connectionEndpointName
		return true
	}

	return false
}

func (s *MediaSubsession) parseSDPAttributeRtpmap(sdpLine string) bool {
	var parseSuccess bool
	var numChannels uint32 = 1

	for {
		if !strings.HasPrefix(sdpLine, "a=rtpmap:") {
			break
		}

		sdpLine = strings.TrimSpace(sdpLine[9:])

		fields := strings.Split(sdpLine, " ")
		if len(fields) != 2 {
			break
		}

		rtpPayloadFormat, err := strconv.Atoi(fields[0])
		if err != nil {
			break
		}
		s.rtpPayloadFormat = uint32(rtpPayloadFormat)

		value := strings.Split(fields[1], "/")
		if len(value) == 2 {
			s.codecName = value[0]

			rtpTimestampFrequency, err := strconv.Atoi(value[1])
			if err != nil {
				break
			}
			s.rtpTimestampFrequency = uint32(rtpTimestampFrequency)
		} else {
			break
		}

		parseSuccess = true
		s.numChannels = numChannels
	}

	return parseSuccess
}

// Check for a "a=control:<control-path>" line:
func (s *MediaSubsession) parseSDPAttributeControl(sdpLine string) bool {
	var parseSuccess bool

	ok := strings.HasPrefix(sdpLine, "a=control:")
	if ok {
		s.controlPath = sdpLine[10:]
		parseSuccess = true
	}

	return parseSuccess
}

func (s *MediaSubsession) parseSDPAttributeRange(sdpLine string) bool {
	var parseSuccess bool

	startTime, endTime, ok := parseRangeAttribute(sdpLine, "npt")
	if ok {
		parseSuccess = true

		playStartTime, _ := strconv.ParseFloat(startTime, 32)
		playEndTime, _ := strconv.ParseFloat(endTime, 32)

		if playStartTime > s.playStartTime {
			s.playStartTime = playStartTime
			if playStartTime > s.parent.maxPlayStartTime {
				s.parent.maxPlayStartTime = playStartTime
			}
		}
		if playEndTime > s.playEndTime {
			s.playEndTime = playEndTime
			if playEndTime > s.parent.maxPlayEndTime {
				s.parent.maxPlayEndTime = playEndTime
			}
		}
	} else if s.absStartTime, s.absEndTime, ok = parseRangeAttribute(sdpLine, "clock"); ok {
		parseSuccess = true
	}

	return parseSuccess
}

func (s *MediaSubsession) parseSDPAttributeFmtp(sdpLine string) bool {
	return true
}

func (s *MediaSubsession) parseSDPAttributeSourceFilter(sdpLine string) bool {
	return parseSourceFilterAttribute(sdpLine)
}

func (s *MediaSubsession) parseSDPAttributeXDimensions(sdpLine string) bool {
	var parseSuccess bool
	var width, height uint
	if n, _ := fmt.Sscanf(sdpLine, "a=x-dimensions:%d,%d", &width, &height); n == 2 {
		s.videoWidth, s.videoHeight = width, height
		parseSuccess = true
	}
	return parseSuccess
}

// check for a "a=framerate: <fps>" r "a=x-framerate: <fps>" line:
func (s *MediaSubsession) parseSDPAttributeFrameRate(sdpLine string) bool {
	parseSuccess := true
	var n int
	for {
		n, _ = fmt.Sscanf(sdpLine, "a=framerate: %f", &s.videoFPS)
		if n == 1 {
			break
		}

		n, _ = fmt.Sscanf(sdpLine, "a=framerate:%f", &s.videoFPS)
		if n == 1 {
			break
		}

		n, _ = fmt.Sscanf(sdpLine, "a=x-framerate: %f", &s.videoFPS)
		if n == 1 {
			break
		}

		parseSuccess = false
		break
	}

	return parseSuccess
}
