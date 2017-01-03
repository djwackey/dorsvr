package livemedia

import (
	"fmt"
	"net"
	"os"

	gs "github.com/djwackey/dorsvr/groupsock"
)

type OnDemandServerMediaSubSession struct {
	ServerMediaSubSession
	cname            string
	sdpLines         string
	portNumForSDP    int
	initialPortNum   uint
	reuseFirstSource bool
	lastStreamToken  *StreamState
	destinations     []*Destinations
	destinationsDict map[string]*Destinations
}

type StreamParameter struct {
	isMulticast     bool
	clientRTPPort   uint
	clientRTCPPort  uint
	serverRTPPort   uint
	serverRTCPPort  uint
	destinationTTL  uint
	destinationAddr string
	streamToken     *StreamState
}

func (s *OnDemandServerMediaSubSession) InitOnDemandServerMediaSubSession(isubsession IServerMediaSubSession) {
	s.initialPortNum = 6970
	s.cname, _ = os.Hostname()
	s.destinationsDict = make(map[string]*Destinations)
	s.InitServerMediaSubSession(isubsession)
}

func (s *OnDemandServerMediaSubSession) SDPLines() string {
	if s.sdpLines == "" {
		inputSource := s.isubsession.createNewStreamSource()

		rtpPayloadType := 96 + s.TrackNumber() - 1

		var dummyAddr string
		dummyGroupSock := gs.NewGroupSock(dummyAddr, 0)
		dummyRTPSink := s.isubsession.createNewRTPSink(dummyGroupSock, rtpPayloadType)

		s.setSDPLinesFromRTPSink(dummyRTPSink, inputSource, 500)
	}

	return s.sdpLines
}

func (s *OnDemandServerMediaSubSession) getStreamParameters(tcpSocketNum net.Conn, destAddr,
	clientSessionID string, clientRTPPort, clientRTCPPort, rtpChannelID, rtcpChannelID uint) *StreamParameter {
	var streamBitrate uint = 500

	sp := new(StreamParameter)

	var rtpPayloadType uint
	if s.lastStreamToken != nil {
		streamState := s.lastStreamToken
		sp.serverRTPPort = streamState.ServerRTPPort()
		sp.serverRTCPPort = streamState.ServerRTCPPort()

		sp.streamToken = s.lastStreamToken
	} else {
		serverPortNum := s.initialPortNum

		sp.serverRTPPort = serverPortNum
		sp.serverRTCPPort = serverPortNum + 1

		var dummyAddr string
		rtpGroupSock := gs.NewGroupSock(dummyAddr, sp.serverRTPPort)
		rtcpGroupSock := gs.NewGroupSock(dummyAddr, sp.serverRTCPPort)

		mediaSource := s.isubsession.createNewStreamSource()
		rtpSink := s.isubsession.createNewRTPSink(rtpGroupSock, rtpPayloadType)

		udpSink := NewBasicUDPSink(rtpGroupSock)

		s.lastStreamToken = NewStreamState(s.isubsession,
			sp.serverRTPPort,
			sp.serverRTCPPort,
			rtpSink,
			udpSink,
			streamBitrate,
			mediaSource,
			rtpGroupSock,
			rtcpGroupSock)
		sp.streamToken = s.lastStreamToken
	}

	dests := NewDestinations(tcpSocketNum, destAddr, clientRTPPort, clientRTCPPort, rtpChannelID, rtcpChannelID)
	s.destinations = append(s.destinations, dests)
	s.destinationsDict[clientSessionID] = dests

	return sp
}

func (s *OnDemandServerMediaSubSession) getAuxSDPLine(rtpSink IRTPSink) string {
	if rtpSink == nil {
		return ""
	}

	return rtpSink.AuxSDPLine()
}

func (s *OnDemandServerMediaSubSession) setSDPLinesFromRTPSink(rtpSink IRTPSink, inputSource IFramedSource, estBitrate uint) {
	if rtpSink == nil {
		return
	}

	mediaType := rtpSink.SdpMediaType()
	rtpmapLine := rtpSink.RtpmapLine()
	rtpPayloadType := rtpSink.RtpPayloadType()

	rangeLine := s.rangeSDPLine()
	auxSDPLine := s.getAuxSDPLine(rtpSink)
	if auxSDPLine == "" {
		auxSDPLine = ""
	}

	ipAddr := "0.0.0.0"

	sdpFmt := "m=%s %d RTP/AVP %d\r\n" +
		"c=IN IP4 %s\r\n" +
		"b=AS:%d\r\n" +
		"%s" +
		"%s" +
		"%s" +
		"a=control:%s\r\n"

	s.sdpLines = fmt.Sprintf(sdpFmt,
		mediaType,
		s.portNumForSDP,
		rtpPayloadType,
		ipAddr,
		estBitrate,
		rtpmapLine,
		rangeLine,
		auxSDPLine,
		s.TrackID())
}

func (s *OnDemandServerMediaSubSession) CNAME() string {
	return s.cname
}

func (s *OnDemandServerMediaSubSession) startStream(clientSessionID uint, streamState *StreamState) (uint, uint) {
	destinations, _ := s.destinationsDict[string(clientSessionID)]
	streamState.startPlaying(destinations)

	fmt.Println("OnDemandServerMediaSubSession::startStream")

	var rtpSeqNum, rtpTimestamp uint
	if streamState.RtpSink() != nil {
		rtpSeqNum = streamState.RtpSink().currentSeqNo()
		rtpTimestamp = streamState.RtpSink().presetNextTimestamp()
	}
	return rtpSeqNum, rtpTimestamp
}

func (s *OnDemandServerMediaSubSession) seekStream() {
	if s.reuseFirstSource {
		return
	}
}

func (s *OnDemandServerMediaSubSession) pauseStream(streamState *StreamState) {
	streamState.pause()
}

func (s *OnDemandServerMediaSubSession) deleteStream(streamState *StreamState) {
	streamState.endPlaying(nil)
}

//////// Destinations ////////
type Destinations struct {
	isTCP         bool
	addrStr       string
	rtpPort       uint
	rtcpPort      uint
	rtpChannelID  uint
	rtcpChannelID uint
	tcpSockNum    net.Conn
}

func NewDestinations(tcpSockNum net.Conn, destAddr string,
	clientRTPPort, clientRTCPPort, rtpChannelID, rtcpChannelID uint) *Destinations {
	destinations := new(Destinations)
	destinations.tcpSockNum = tcpSockNum
	destinations.addrStr = destAddr
	destinations.rtpPort = clientRTPPort
	destinations.rtcpPort = clientRTCPPort
	destinations.rtpChannelID = rtpChannelID
	destinations.rtcpChannelID = rtcpChannelID
	return destinations
}
