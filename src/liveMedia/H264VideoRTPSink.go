package liveMedia

import (
	. "groupsock"
)

type H264VideoRTPSink struct {
	VideoRTPSink
}

func NewH264VideoRTPSink(rtpGroupSock *GroupSock, rtpPayloadType int) *H264VideoRTPSink {
	h264VideoRTPSink := new(H264VideoRTPSink)
	h264VideoRTPSink.InitVideoRTPSink()
	return h264VideoRTPSink
}
