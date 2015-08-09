package liveMedia

import (
	"fmt"
	. "groupsock"
	"net"
	"os"
)

type OnDemandServerMediaSubSession struct {
	ServerMediaSubSession
	cname           string
	sdpLines        string
	portNumForSDP   int
	initialPortNum  uint
	lastStreamToken *StreamState
	destinations    []*Destinations
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
	this.InitServerMediaSubSession(isubsession)
}

func (this *OnDemandServerMediaSubSession) SDPLines() string {
	if this.sdpLines == "" {
		inputSource := this.isubsession.createNewStreamSource()

		rtpPayloadType := 96 + this.TrackNumber() - 1
		dummyGroupSock := NewGroupSock(0)
		dummyRTPSink := this.isubsession.createNewRTPSink(dummyGroupSock, rtpPayloadType)

		this.setSDPLinesFromRTPSink(dummyRTPSink, inputSource, 500)
	}

	return this.sdpLines
}

func (this *OnDemandServerMediaSubSession) getStreamParameters(tcpSocketNum *net.Conn, clientRTPPort, clientRTCPPort, rtpChannelId, rtcpChannelId int) *StreamParameter {
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

		rtpGroupSock := NewGroupSock(sp.serverRTPPort)
		rtcpGroupSock := NewGroupSock(sp.serverRTCPPort)

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

	//var destAddr string
	//dests := NewDestinations(tcpSocketNum, destAddr, clientRTPPort, clientRTCPPort, rtpChannelId, rtcpChannelId)
	//append(this.destinations, dests)

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
		this.TrackId())
}

func (this *OnDemandServerMediaSubSession) CNAME() string {
	return this.cname
}

func (this *OnDemandServerMediaSubSession) startStream(streamState *StreamState) {
	streamState.startPlaying()
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
	addr          string
	rtpPort       int
	rtcpPort      int
	rtpChannelId  int
	rtcpChannelId int
	tcpSockNum    *net.Conn
}

func NewDestinations(tcpSockNum *net.Conn, destAddr string, clientRTPPort, clientRTCPPort, rtpChannelId, rtcpChannelId int) *Destinations {
	destinations := new(Destinations)
	destinations.tcpSockNum = tcpSockNum
	destinations.addr = destAddr
	destinations.rtpPort = clientRTPPort
	destinations.rtcpPort = clientRTCPPort
	destinations.rtpChannelId = rtpChannelId
	destinations.rtcpChannelId = rtcpChannelId
	return destinations
}
