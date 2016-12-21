package rtspserver

import (
	"fmt"
	"net"
	"os"
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

func (this *OnDemandServerMediaSubSession) InitOnDemandServerMediaSubSession(isubsession IServerMediaSubSession) {
	this.initialPortNum = 6970
	this.cname, _ = os.Hostname()
	this.destinationsDict = make(map[string]*Destinations)
	this.InitServerMediaSubSession(isubsession)
}

func (this *OnDemandServerMediaSubSession) SDPLines() string {
	if this.sdpLines == "" {
		inputSource := this.isubsession.createNewStreamSource()

		rtpPayloadType := 96 + this.TrackNumber() - 1

		var dummyAddr string
		dummyGroupSock := NewGroupSock(dummyAddr, 0)
		dummyRTPSink := this.isubsession.createNewRTPSink(dummyGroupSock, rtpPayloadType)

		this.setSDPLinesFromRTPSink(dummyRTPSink, inputSource, 500)
	}

	return this.sdpLines
}

func (this *OnDemandServerMediaSubSession) getStreamParameters(tcpSocketNum net.Conn, destAddr,
	clientSessionID string, clientRTPPort, clientRTCPPort, rtpChannelID, rtcpChannelID uint) *StreamParameter {
	var streamBitrate uint = 500

	sp := new(StreamParameter)

	var rtpPayloadType uint
	if this.lastStreamToken != nil {
		streamState := this.lastStreamToken
		sp.serverRTPPort = streamState.ServerRTPPort()
		sp.serverRTCPPort = streamState.ServerRTCPPort()

		sp.streamToken = this.lastStreamToken
	} else {
		serverPortNum := this.initialPortNum

		sp.serverRTPPort = serverPortNum
		sp.serverRTCPPort = serverPortNum + 1

		var dummyAddr string
		rtpGroupSock := NewGroupSock(dummyAddr, sp.serverRTPPort)
		rtcpGroupSock := NewGroupSock(dummyAddr, sp.serverRTCPPort)

		mediaSource := this.isubsession.createNewStreamSource()
		rtpSink := this.isubsession.createNewRTPSink(rtpGroupSock, rtpPayloadType)

		udpSink := NewBasicUDPSink(rtpGroupSock)

		this.lastStreamToken = NewStreamState(this.isubsession,
			sp.serverRTPPort,
			sp.serverRTCPPort,
			rtpSink,
			udpSink,
			streamBitrate,
			mediaSource,
			rtpGroupSock,
			rtcpGroupSock)
		sp.streamToken = this.lastStreamToken
	}

	dests := NewDestinations(tcpSocketNum, destAddr, clientRTPPort, clientRTCPPort, rtpChannelID, rtcpChannelID)
	this.destinations = append(this.destinations, dests)
	this.destinationsDict[clientSessionID] = dests

	return sp
}

func (this *OnDemandServerMediaSubSession) getAuxSDPLine(rtpSink IRTPSink) string {
	if rtpSink == nil {
		return ""
	}

	return rtpSink.AuxSDPLine()
}

func (this *OnDemandServerMediaSubSession) setSDPLinesFromRTPSink(rtpSink IRTPSink, inputSource IFramedSource, estBitrate uint) {
	if rtpSink == nil {
		return
	}

	mediaType := rtpSink.SdpMediaType()
	rtpmapLine := rtpSink.RtpmapLine()
	rtpPayloadType := rtpSink.RtpPayloadType()

	rangeLine := this.rangeSDPLine()
	auxSDPLine := this.getAuxSDPLine(rtpSink)
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

	this.sdpLines = fmt.Sprintf(sdpFmt,
		mediaType,
		this.portNumForSDP,
		rtpPayloadType,
		ipAddr,
		estBitrate,
		rtpmapLine,
		rangeLine,
		auxSDPLine,
		this.TrackID())
}

func (this *OnDemandServerMediaSubSession) CNAME() string {
	return this.cname
}

func (this *OnDemandServerMediaSubSession) startStream(clientSessionId uint, streamState *StreamState) (uint, uint) {
	destinations, _ := this.destinationsDict[string(clientSessionId)]
	streamState.startPlaying(destinations)

	fmt.Println("OnDemandServerMediaSubSession::startStream")

	var rtpSeqNum, rtpTimestamp uint
	if streamState.RtpSink() != nil {
		rtpSeqNum = streamState.RtpSink().currentSeqNo()
		rtpTimestamp = streamState.RtpSink().presetNextTimestamp()
	}
	return rtpSeqNum, rtpTimestamp
}

func (this *OnDemandServerMediaSubSession) seekStream() {
	if this.reuseFirstSource {
		return
	}
}

func (this *OnDemandServerMediaSubSession) pauseStream(streamState *StreamState) {
	streamState.pause()
}

func (this *OnDemandServerMediaSubSession) deleteStream(streamState *StreamState) {
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
