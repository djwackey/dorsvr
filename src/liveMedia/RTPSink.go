package liveMedia

import (
	"fmt"
	. "groupsock"
)

type RTPSink struct {
	MediaSink
	ssrc                  uint
	seqNo                 uint
	rtpPayloadType        uint
	rtpTimestampFrequency uint
	rtpPayloadFormatName  string
	rtpInterface          *RTPInterface
}

type IRTPSink interface {
	RtpPayloadType() uint
	RtpmapLine() string
	SdpMediaType() string
	startPlaying(source IFramedSource) bool
	stopPlaying()
	continuePlaying()
}

func (this *RTPSink) InitRTPSink(rtpSink IRTPSink, gs *GroupSock, rtpPayloadType, rtpTimestampFrequency uint, rtpPayloadFormatName string) {
	this.InitMediaSink(rtpSink)
	this.rtpInterface = NewRTPInterface(this, gs)
	this.rtpPayloadType = rtpPayloadType
	this.rtpTimestampFrequency = rtpTimestampFrequency
	this.rtpPayloadFormatName = rtpPayloadFormatName
}

func (this *RTPSink) SSRC() uint {
	return this.ssrc
}

func (this *RTPSink) AuxSDPLine() string {
	return ""
}

func (this *RTPSink) SdpMediaType() string {
	return "data"
}

func (this *RTPSink) RtpPayloadType() uint {
	return this.rtpPayloadType
}

func (this *RTPSink) RtpmapLine() string {
	var rtpmapLine string
	if this.rtpPayloadType >= 96 {
		encodingParamsPart := ""
		rtpmapFmt := "a=rtpmap:%d %s/%d%s\r\n"
		rtpmapLine = fmt.Sprintf(rtpmapFmt,
			this.RtpPayloadType(),
			this.RtpPayloadFormatName(),
			this.RtpTimestampFrequency(), encodingParamsPart)
	}

	return rtpmapLine
}

func (this *RTPSink) RtpPayloadFormatName() string {
	return this.rtpPayloadFormatName
}

func (this *RTPSink) RtpTimestampFrequency() uint {
	return this.rtpTimestampFrequency
}
