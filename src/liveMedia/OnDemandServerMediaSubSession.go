package liveMedia

import (
	"fmt"
	. "groupsock"
)

type OnDemandServerMediaSubSession struct {
    ServerMediaSubSession
	SDPLines      string
	trackId       string
	trackNumber   int
	portNumForSDP int
    initialPortNum int
    lastStreamToken interface{}
}

type StreamParameter struct {
    isMulticast bool
    clientRTPPort int
    clientRTCPPort int
    serverRTPPort int
    serverRTCPPort int
    destinationTTL uint
    destinationAddr string
    streamToken interface{}
}

func (this *OnDemandServerMediaSubSession) InitOnDemandServerMediaSubSession() {
}

func (this *OnDemandServerMediaSubSession) sdpLines() {
	if this.SDPLines != "" {
		this.setSDPLinesFromRTPSink(nil, 500)
	}
}

func (this *OnDemandServerMediaSubSession) getStreamParameters(rtpChannelId, rtcpChannelId uint) *StreamParameter {
    streamBitrate = 500
    serverPortNum = this.initialPortNum

    serverRTPPort  := serverPortNum
    serverRTCPPort := serverPortNum + 1

    rtpGroupSock  := NewGroupSock(serverRTPPort)
    rtcpGroupSock := NewGroupSock(serverRTCPPort)

    udpSink := NewBasicUDPSink(rtpGroupSock)

    this.lastStreamToken = NewStreamState(serverRTPPort, serverRTCPPort, rtpSink, udpSink, streamBitrate, mediaSource, rtpGroupSock, rtcpGroupSock)

    sp := new(StreamParameter)
    return sp
}

func (this *OnDemandServerMediaSubSession) TrackId() string {
	if this.trackId == "" {
		this.trackId = fmt.Sprintf("track%d", this.trackNumber)
	}
	return this.trackId
}

func (this *OnDemandServerMediaSubSession) setSDPLinesFromRTPSink(rtpSink *RTPSink, estBitrate uint) {
	if rtpSink == nil {
		return
	}

	rtpPayloadType := rtpSink.RtpPayloadType()
	mediaType := rtpSink.SdpMediaType()
	rangeLine := this.rangeSDPLine()

	auxSDPLine := this.getAuxSDPLine()
	if auxSDPLine == "" {
		auxSDPLine = ""
	}

	ipAddr, _ := OurIPAddress()

	sdpFmt := "m=%s %u RTP/AVP %d\r\n" +
		"c=IN IP4 %s\r\n" +
		"b=AS:%u\r\n" +
		"%s" +
		"%s" +
		"%s" +
		"a=control:%s\r\n"

	this.SDPLines = fmt.Sprintf(sdpFmt, mediaType, this.portNumForSDP, rtpPayloadType, ipAddr, estBitrate, rangeLine, auxSDPLine, this.TrackId())
}

func (this *OnDemandServerMediaSubSession) getAuxSDPLine(rtpSink *RTPSink) string {
    if rtpSink == nil {
	    return ""
    } else {
        return rtpSink.AuxSDPLine()
    }
}

func (this *OnDemandServerMediaSubSession) startStream() {
    streamState.startPlaying()
}

func (this *OnDemandServerMediaSubSession) pauseStream() {
    streamState.pause()
}

func (this *OnDemandServerMediaSubSession) deleteStream() {
    streamState.endPlaying()
}
