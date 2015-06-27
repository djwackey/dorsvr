package liveMedia

import (
	"fmt"
	. "groupsock"
)

type OnDemandServerMediaSubSession struct {
	SDPLines      string
	trackId       string
	trackNumber   int
	portNumForSDP int
}

func (this *OnDemandServerMediaSubSession) sdpLines() {
	if this.SDPLines != "" {
		//this.setSDPLinesFromRTPSink()
	}
}

func (this *OnDemandServerMediaSubSession) getStreamParameters(rtpChannelId, rtcpChannelId uint) {
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

func (this *OnDemandServerMediaSubSession) rangeSDPLine() string {
	return ""
}

func (this *OnDemandServerMediaSubSession) getAuxSDPLine() string {
	return ""
}
