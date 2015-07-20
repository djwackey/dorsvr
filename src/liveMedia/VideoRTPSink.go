package liveMedia

import (
	. "groupsock"
)

type VideoRTPSink struct {
	MultiFramedRTPSink
}

func (this *VideoRTPSink) InitVideoRTPSink(rtpGroupSock *GroupSock, rtpPayloadType int,
	rtpTimestampFrequency uint,
	rtpPayloadFormatName string) {
	this.InitMultiFramedRTPSink(rtpGroupSock, rtpPayloadType, rtpTimestampFrequency, rtpPayloadFormatName)
}

func (this *VideoRTPSink) SdpMediaType() string {
	return "video"
}
