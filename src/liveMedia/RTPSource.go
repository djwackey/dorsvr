package liveMedia

import (
	. "groupsock"
)

type RTPSource struct {
	FramedSource
	SSRC                   uint
	lastReceivedSSRC       uint
	rtpPayloadFormat       uint
	timestampFrequency     uint
	curPacketSyncUsingRTCP bool
	receptionStatsDB       *RTPReceptionStatsDB
	rtpInterface           *RTPInterface
}

func NewRTPSource() *RTPSource {
	source := new(RTPSource)
	return source
}

func (source *RTPSource) InitRTPSouce(isource IFramedSource, RTPgs *GroupSock,
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
