package liveMedia

import (
	. "groupsock"
)

type VideoRTPSink struct {
	MultiFramedRTPSink
}

func (this *VideoRTPSink) InitVideoRTPSink(rtpSink IRTPSink, rtpGroupSock *GroupSock, rtpPayloadType, rtpTimestampFrequency uint, rtpPayloadFormatName string) {
	this.InitMultiFramedRTPSink(rtpSink, rtpGroupSock, rtpPayloadType, rtpTimestampFrequency, rtpPayloadFormatName)
}

func (this *VideoRTPSink) SdpMediaType() string {
	return "video"
}
