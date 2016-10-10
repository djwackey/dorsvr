package liveMedia

import (
	. "groupsock"
)

type SimpleRTPSink struct {
	MultiFramedRTPSink
	allowMultipleFramesPerPacket bool
}

func NewSimpleRTPSink(rtpGS *GroupSock, rtpPayloadFormat,
	rtpTimestampFrequency, numChannels uint,
	sdpMediaTypeString, rtpPayloadFormatName string,
	allowMultipleFramesPerPacket, doNormalMBitRule bool) *SimpleRTPSink {
	simpleRTPSink := new(SimpleRTPSink)
	simpleRTPSink.InitMultiFramedRTPSink(simpleRTPSink, rtpGS, rtpPayloadFormat, rtpTimestampFrequency, rtpPayloadFormatName)
	simpleRTPSink.allowMultipleFramesPerPacket = allowMultipleFramesPerPacket
	return simpleRTPSink
}

func (this *SimpleRTPSink) AuxSDPLine() string {
	return ""
}

func (this *SimpleRTPSink) continuePlaying() {
}
