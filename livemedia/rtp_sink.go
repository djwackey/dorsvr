package livemedia

import (
	"fmt"
	"net"
	sys "syscall"

	gs "github.com/djwackey/dorsvr/groupsock"
	//"github.com/djwackey/dorsvr/log"
)

//////// RTPSink ////////
type RTPSink struct {
	MediaSink
	seqNo                       uint32
	_ssrc                       uint32
	_octetCount                 uint
	_packetCount                uint // incl RTP hdr
	totalOctetCount             uint
	timestampBase               uint32
	_rtpPayloadType             uint32
	rtpTimestampFrequency       uint32
	rtpPayloadFormatName        string
	_enableRTCPReports          bool
	_nextTimestampHasBeenPreset bool
	_transmissionStatsDB        *RTPTransmissionStatsDB
	rtpInterface                *RTPInterface
}

func (s *RTPSink) InitRTPSink(rtpSink IMediaSink, g *gs.GroupSock, rtpPayloadType,
	rtpTimestampFrequency uint32, rtpPayloadFormatName string) {
	s.InitMediaSink(rtpSink)
	s.rtpInterface = newRTPInterface(s, g)
	s.rtpPayloadFormatName = rtpPayloadFormatName
	s.rtpTimestampFrequency = rtpTimestampFrequency
	s._rtpPayloadType = rtpPayloadType
	s._nextTimestampHasBeenPreset = true
	s._transmissionStatsDB = newRTPTransmissionStatsDB(s)

	s.seqNo = gs.OurRandom16()
	s._ssrc = gs.OurRandom32()
	s.timestampBase = gs.OurRandom32()
}

func (s *RTPSink) addStreamSocket(socketNum net.Conn, streamChannelID uint) {
	s.rtpInterface.addStreamSocket(socketNum, streamChannelID)
}

func (s *RTPSink) delStreamSocket(socketNum net.Conn, streamChannelID uint) {
	s.rtpInterface.delStreamSocket(socketNum, streamChannelID)
}

func (s *RTPSink) currentSeqNo() uint32 {
	return s.seqNo
}

func (s *RTPSink) sdpMediaType() string {
	return "data"
}

func (s *RTPSink) rtpPayloadType() uint32 {
	return s._rtpPayloadType
}

func (s *RTPSink) rtpmapLine() (line string) {
	var encodingParamsPart string
	if s._rtpPayloadType >= 96 {
		line = fmt.Sprintf("a=rtpmap:%d %s/%d%s\r\n",
			s._rtpPayloadType,
			s.rtpPayloadFormatName,
			s.rtpTimestampFrequency, encodingParamsPart)
	}
	return
}

func (s *RTPSink) ssrc() uint32 {
	return s._ssrc
}

func (s *RTPSink) octetCount() uint {
	return s._octetCount
}

func (s *RTPSink) packetCount() uint {
	return s._packetCount
}

func (s *RTPSink) enableRTCPReports() bool {
	return s._enableRTCPReports
}

func (s *RTPSink) nextTimestampHasBeenPreset() bool {
	return s._nextTimestampHasBeenPreset
}

func (s *RTPSink) transmissionStatsDB() *RTPTransmissionStatsDB {
	return s._transmissionStatsDB
}

func (s *RTPSink) presetNextTimestamp() uint32 {
	var timeNow sys.Timeval
	sys.Gettimeofday(&timeNow)

	tsNow := s.convertToRTPTimestamp(timeNow)
	s.timestampBase = tsNow
	s._nextTimestampHasBeenPreset = true

	return tsNow
}

func (s *RTPSink) convertToRTPTimestamp(tv sys.Timeval) uint32 {
	// Begin by converting from "struct timeval" units to RTP timestamp units:
	timestampIncrement := s.rtpTimestampFrequency * uint32(tv.Sec)
	timestampIncrement += (2.0*s.rtpTimestampFrequency*uint32(tv.Usec) + 1000000.0) / 2000000

	// Then add this to our 'timestamp base':
	if s._nextTimestampHasBeenPreset {
		// Make the returned timestamp the same as the current "fTimestampBase",
		// so that timestamps begin with the value that was previously preset:
		s.timestampBase -= timestampIncrement
		s._nextTimestampHasBeenPreset = false
	}

	// return RTP Timestamp
	return s.timestampBase + timestampIncrement
}

func (s *RTPSink) setServerRequestAlternativeByteHandler(socketNum net.Conn, handler interface{}) {
	s.rtpInterface.setServerRequestAlternativeByteHandler(socketNum, handler)
}
