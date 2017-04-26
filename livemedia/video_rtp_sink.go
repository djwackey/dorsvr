package livemedia

import gs "github.com/djwackey/dorsvr/groupsock"

type VideoRTPSink struct {
	MultiFramedRTPSink
}

func (s *VideoRTPSink) initVideoRTPSink(rtpSink IMediaSink, rtpGroupSock *gs.GroupSock,
	rtpPayloadType, rtpTimestampFrequency uint32, rtpPayloadFormatName string) {
	s.InitMultiFramedRTPSink(rtpSink, rtpGroupSock, rtpPayloadType,
		rtpTimestampFrequency, rtpPayloadFormatName)
}

func (s *VideoRTPSink) sdpMediaType() string {
	return "video"
}
