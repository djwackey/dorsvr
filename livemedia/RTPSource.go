package livemedia

import gs "github.com/djwackey/dorsvr/groupsock"

type RTPSource struct {
	FramedSource
	SSRC                   uint32
	lastReceivedSSRC       uint
	rtpPayloadFormat       uint
	timestampFrequency     uint
	curPacketRTPSeqNum     uint32
	curPacketRTPTimestamp  uint32
	curPacketSyncUsingRTCP bool
	curPacketMarkerBit     bool
	receptionStatsDB       *RTPReceptionStatsDB
	rtpInterface           *RTPInterface
}

func NewRTPSource() *RTPSource {
	source := new(RTPSource)
	return source
}

func (source *RTPSource) InitRTPSouce(isource IFramedSource, RTPgs *gs.GroupSock,
	rtpPayloadFormat, rtpTimestampFrequency uint) {
	source.rtpInterface = NewRTPInterface(source, RTPgs)
	source.lastReceivedSSRC = 0
	source.rtpPayloadFormat = rtpPayloadFormat
	source.timestampFrequency = rtpTimestampFrequency
	source.SSRC = OurRandom32()
	source.curPacketSyncUsingRTCP = false
	source.receptionStatsDB = NewRTPReceptionStatsDB()
	source.InitFramedSource(isource)
}

func (source *RTPSource) RTPPayloadFormat() uint {
	return source.rtpPayloadFormat
}

func (source *RTPSource) TimestampFrequency() uint {
	return source.timestampFrequency
}

func (source *RTPSource) setStreamSocket() {
}

func (source *RTPSource) ReceptionStatsDB() *RTPReceptionStatsDB {
	return source.receptionStatsDB
}
