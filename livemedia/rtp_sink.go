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
	ssrc                       uint32
	seqNo                      uint32
	octetCount                 uint
	packetCount                uint // incl RTP hdr
	totalOctetCount            uint
	timestampBase              uint32
	rtpPayloadType             uint32
	rtpTimestampFrequency      uint32
	rtpPayloadFormatName       string
	enableRTCPReports          bool
	nextTimestampHasBeenPreset bool
	rtpInterface               *RTPInterface
	transmissionStatsDB        *RTPTransmissionStatsDB
}

func (s *RTPSink) InitRTPSink(rtpSink IMediaSink, g *gs.GroupSock, rtpPayloadType,
	rtpTimestampFrequency uint32, rtpPayloadFormatName string) {
	s.initMediaSink(rtpSink)
	s.rtpInterface = newRTPInterface(s, g)
	s.rtpPayloadType = rtpPayloadType
	s.rtpTimestampFrequency = rtpTimestampFrequency
	s.rtpPayloadFormatName = rtpPayloadFormatName

	s.seqNo = gs.OurRandom16()
	s.ssrc = gs.OurRandom32()
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

func (s *RTPSink) SdpMediaType() string {
	return "data"
}

func (s *RTPSink) RtpPayloadType() uint32 {
	return s.rtpPayloadType
}

func (s *RTPSink) RtpmapLine() string {
	var rtpmapLine, encodingParamsPart string
	if s.rtpPayloadType >= 96 {
		rtpmapLine = fmt.Sprintf("a=rtpmap:%d %s/%d%s\r\n",
			s.rtpPayloadType,
			s.rtpPayloadFormatName,
			s.rtpTimestampFrequency, encodingParamsPart)
	}

	return rtpmapLine
}

func (s *RTPSink) presetNextTimestamp() uint32 {
	var timeNow sys.Timeval
	sys.Gettimeofday(&timeNow)

	tsNow := s.convertToRTPTimestamp(timeNow)
	s.timestampBase = tsNow
	s.nextTimestampHasBeenPreset = true

	return tsNow
}

func (s *RTPSink) convertToRTPTimestamp(tv sys.Timeval) uint32 {
	// Begin by converting from "struct timeval" units to RTP timestamp units:
	timestampIncrement := s.rtpTimestampFrequency * uint32(tv.Sec)
	timestampIncrement += (2.0*s.rtpTimestampFrequency*uint32(tv.Usec) + 1000000.0) / 2000000

	// Then add this to our 'timestamp base':
	if s.nextTimestampHasBeenPreset {
		// Make the returned timestamp the same as the current "fTimestampBase",
		// so that timestamps begin with the value that was previously preset:
		s.timestampBase -= timestampIncrement
		s.nextTimestampHasBeenPreset = false
	}

	// return RTP Timestamp
	return s.timestampBase + timestampIncrement
}

func (s *RTPSink) setServerRequestAlternativeByteHandler(socketNum net.Conn, handler interface{}) {
	s.rtpInterface.setServerRequestAlternativeByteHandler(socketNum, handler)
}

//////// RTPTransmissionStatsDB ////////
type RTPTransmissionStatsDB struct {
}

//////// RTPTransmissionStats ////////
type RTPTransmissionStats struct {
	isPacket              bool
	SSRC                  uint
	jitter                uint
	packetLossRatio       uint
	totNumPacketsLost     uint
	lastPacketNumReceived uint
	timeCreated           sys.Timeval
	timeReceived          sys.Timeval
}
