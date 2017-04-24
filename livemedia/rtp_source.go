package livemedia

import gs "github.com/djwackey/dorsvr/groupsock"

type RTPSource struct {
	FramedSource
	ssrc                   uint32
	lastReceivedSSRC       uint32
	rtpPayloadFormat       uint32
	timestampFrequency     uint32
	curPacketRTPSeqNum     uint32
	curPacketRTPTimestamp  uint32
	curPacketSyncUsingRTCP bool
	curPacketMarkerBit     bool
	receptionStatsDB       *RTPReceptionStatsDB
	rtpInterface           *RTPInterface
}

func newRTPSource() *RTPSource {
	return &RTPSource{}
}

func (s *RTPSource) initRTPSouce(isource IFramedSource, RTPgs *gs.GroupSock,
	rtpPayloadFormat, rtpTimestampFrequency uint32) {
	s.rtpInterface = newRTPInterface(s, RTPgs)
	s.lastReceivedSSRC = 0
	s.rtpPayloadFormat = rtpPayloadFormat
	s.timestampFrequency = rtpTimestampFrequency
	s.ssrc = gs.OurRandom32()
	s.curPacketSyncUsingRTCP = false
	s.receptionStatsDB = newRTPReceptionStatsDB()
	s.initFramedSource(isource)
}

func (s *RTPSource) SetStreamSocket() {
}
