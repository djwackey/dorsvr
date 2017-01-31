package livemedia

import (
	"fmt"
	"net"
	sys "syscall"

	gs "github.com/djwackey/dorsvr/groupsock"
)

//////// RTPSink ////////
type IRTPSink interface {
	RtpPayloadType() uint
	AuxSDPLine() string
	RtpmapLine() string
	SdpMediaType() string
	currentSeqNo() uint
	StartPlaying(source IFramedSource) bool
	StopPlaying()
	ContinuePlaying()
	addStreamSocket(sockNum net.Conn, streamChannelID uint)
	delStreamSocket()
	presetNextTimestamp() uint
}

type RTPSink struct {
	MediaSink
	ssrc                       uint
	seqNo                      uint
	octetCount                 uint
	packetCount                uint // incl RTP hdr
	timestampBase              uint
	totalOctetCount            uint
	rtpPayloadType             uint
	rtpTimestampFrequency      uint
	timestampFrequency         uint
	rtpPayloadFormatName       string
	enableRTCPReports          bool
	nextTimestampHasBeenPreset bool
	rtpInterface               *RTPInterface
	transmissionStatsDB        *RTPTransmissionStatsDB
}

func (s *RTPSink) InitRTPSink(rtpSink IRTPSink, gs *gs.GroupSock, rtpPayloadType,
	rtpTimestampFrequency uint, rtpPayloadFormatName string) {
	s.InitMediaSink(rtpSink)
	s.rtpInterface = NewRTPInterface(s, gs)
	s.rtpPayloadType = rtpPayloadType
	s.rtpTimestampFrequency = rtpTimestampFrequency
	s.rtpPayloadFormatName = rtpPayloadFormatName
}

func (s *RTPSink) SSRC() uint {
	return s.ssrc
}

func (s *RTPSink) addStreamSocket(sockNum net.Conn, streamChannelID uint) {
	s.rtpInterface.addStreamSocket(sockNum, streamChannelID)
}

func (s *RTPSink) delStreamSocket() {
	s.rtpInterface.delStreamSocket()
}

func (s *RTPSink) currentSeqNo() uint {
	return s.seqNo
}

func (sink *RTPSink) SdpMediaType() string {
	return "data"
}

func (sink *RTPSink) RtpPayloadType() uint {
	return sink.rtpPayloadType
}

func (sink *RTPSink) RtpmapLine() string {
	var rtpmapLine string
	if sink.rtpPayloadType >= 96 {
		encodingParamsPart := ""
		rtpmapFmt := "a=rtpmap:%d %s/%d%s\r\n"
		rtpmapLine = fmt.Sprintf(rtpmapFmt,
			sink.rtpPayloadType,
			sink.rtpPayloadFormatName,
			sink.rtpTimestampFrequency, encodingParamsPart)
	}

	return rtpmapLine
}

func (s *RTPSink) presetNextTimestamp() uint {
	var timeNow sys.Timeval
	sys.Gettimeofday(&timeNow)

	tsNow := s.convertToRTPTimestamp(timeNow)
	s.timestampBase = tsNow
	s.nextTimestampHasBeenPreset = true

	return tsNow
}

func (s *RTPSink) convertToRTPTimestamp(tv sys.Timeval) uint {
	// Begin by converting from "struct timeval" units to RTP timestamp units:
	timestampIncrement := s.timestampFrequency * uint(tv.Sec)
	timestampIncrement += (2.0*s.timestampFrequency*uint(tv.Usec) + 1000000.0) / 2000000

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
