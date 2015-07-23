package liveMedia

import (
	"fmt"
	. "groupsock"
	"os"
)

type OnDemandServerMediaSubSession struct {
	ServerMediaSubSession
	cname           string
	sdpLines        string
	portNumForSDP   int
	initialPortNum  uint
	lastStreamToken *StreamState
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
		inputSource := this.isubsession.CreateNewStreamSource()

		rtpPayloadType := 96 + this.TrackNumber() - 1
		dummyGroupSock := NewGroupSock(0)
		dummyRTPSink := this.isubsession.CreateNewRTPSink(dummyGroupSock, rtpPayloadType)

		this.setSDPLinesFromRTPSink(dummyRTPSink, inputSource, 500)
	}

	return this.sdpLines
}

func (this *OnDemandServerMediaSubSession) getStreamParameters(rtpChannelId, rtcpChannelId int) *StreamParameter {
	var streamBitrate uint = 500

	sp := new(StreamParameter)

	var serverRTPPort, serverRTCPPort, rtpPayloadType uint
	if this.lastStreamToken != nil {
		streamState := this.lastStreamToken
		serverRTPPort = streamState.ServerRTPPort()
		serverRTCPPort = streamState.ServerRTCPPort()

		sp.streamToken = this.lastStreamToken
	} else {
		serverPortNum := this.initialPortNum

		serverRTPPort = serverPortNum
		serverRTCPPort = serverPortNum + 1

		rtpGroupSock := NewGroupSock(serverRTPPort)
		rtcpGroupSock := NewGroupSock(serverRTCPPort)

		mediaSource := this.isubsession.CreateNewStreamSource()
		rtpSink := this.isubsession.CreateNewRTPSink(rtpGroupSock, rtpPayloadType)

		udpSink := NewBasicUDPSink(rtpGroupSock)

		this.lastStreamToken = NewStreamState(this.isubsession,
			serverRTPPort,
			serverRTCPPort,
			rtpSink,
			udpSink,
			streamBitrate,
			mediaSource,
			rtpGroupSock,
			rtcpGroupSock)
		sp.streamToken = this.lastStreamToken
	}

	return sp
}

func (this *OnDemandServerMediaSubSession) setSDPLinesFromRTPSink(rtpSink IRTPSink, inputSource IFramedSource, estBitrate uint) {
	if rtpSink == nil {
		return
	}

	rtpPayloadType := rtpSink.RtpPayloadType()
	mediaType := rtpSink.SdpMediaType()
	rtpmapLine := rtpSink.RtpmapLine()
	rangeLine := "" //this.rangeSDPLine()

	auxSDPLine := "" //this.getAuxSDPLine()
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

func (this *OnDemandServerMediaSubSession) getAuxSDPLine(rtpSink *RTPSink) string {
	if rtpSink == nil {
		return ""
	} else {
		return rtpSink.AuxSDPLine()
	}
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
	streamState.endPlaying()
}
