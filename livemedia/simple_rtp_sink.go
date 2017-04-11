package livemedia

import gs "github.com/djwackey/dorsvr/groupsock"

type SimpleRTPSink struct {
	MultiFramedRTPSink
	allowMultipleFramesPerPacket bool
}

func newSimpleRTPSink(rtpGS *gs.GroupSock, rtpPayloadFormat,
	rtpTimestampFrequency, numChannels uint32,
	sdpMediaTypeString, rtpPayloadFormatName string,
	allowMultipleFramesPerPacket, doNormalMBitRule bool) *SimpleRTPSink {
	sink := new(SimpleRTPSink)
	sink.InitMultiFramedRTPSink(sink, rtpGS, rtpPayloadFormat, rtpTimestampFrequency, rtpPayloadFormatName)
	sink.allowMultipleFramesPerPacket = allowMultipleFramesPerPacket
	return sink
}

func (s *SimpleRTPSink) AuxSDPLine() string {
	return ""
}

func (s *SimpleRTPSink) ContinuePlaying() {
}
