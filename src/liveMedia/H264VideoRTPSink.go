package liveMedia

import (
	. "groupsock"
)

type H264VideoRTPSink struct {
	VideoRTPSink
	SPS int
	PPS int
}

func NewH264VideoRTPSink(rtpGroupSock *GroupSock, rtpPayloadType int) *H264VideoRTPSink {
	h264VideoRTPSink := new(H264VideoRTPSink)
	h264VideoRTPSink.InitVideoRTPSink(rtpGroupSock, rtpPayloadType, 90000, "H264")
	return h264VideoRTPSink
}
