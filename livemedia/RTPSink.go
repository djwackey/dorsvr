package livemedia

import (
	"fmt"
	gs "github.com/djwackey/dorsvr/groupsock"
	"github.com/djwackey/dorsvr/utils"
	"net"
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

func (sink *RTPSink) InitRTPSink(rtpSink IRTPSink, gs *gs.GroupSock, rtpPayloadType,
	rtpTimestampFrequency uint, rtpPayloadFormatName string) {
	sink.InitMediaSink(rtpSink)
	sink.rtpInterface = NewRTPInterface(sink, gs)
	sink.rtpPayloadType = rtpPayloadType
	sink.rtpTimestampFrequency = rtpTimestampFrequency
	sink.rtpPayloadFormatName = rtpPayloadFormatName
}

func (sink *RTPSink) SSRC() uint {
	return sink.ssrc
}

func (sink *RTPSink) addStreamSocket(sockNum net.Conn, streamChannelID uint) {
	sink.rtpInterface.addStreamSocket(sockNum, streamChannelID)
}

func (sink *RTPSink) delStreamSocket() {
	sink.rtpInterface.delStreamSocket()
}

func (sink *RTPSink) currentSeqNo() uint {
	return sink.seqNo
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
			sink.RtpPayloadType(),
			sink.RtpPayloadFormatName(),
			sink.RtpTimestampFrequency(), encodingParamsPart)
	}

	return rtpmapLine
}

func (sink *RTPSink) RtpPayloadFormatName() string {
	return sink.rtpPayloadFormatName
}

func (sink *RTPSink) RtpTimestampFrequency() uint {
	return sink.rtpTimestampFrequency
}

func (sink *RTPSink) presetNextTimestamp() uint {
	var timeNow utils.Timeval
	utils.GetTimeOfDay(&timeNow)

	tsNow := sink.convertToRTPTimestamp(timeNow)
	sink.timestampBase = tsNow
	sink.nextTimestampHasBeenPreset = true

	return tsNow
}

func (sink *RTPSink) convertToRTPTimestamp(tv utils.Timeval) uint {
	// Begin by converting from "struct timeval" units to RTP timestamp units:
	timestampIncrement := sink.timestampFrequency * uint(tv.Tv_sec)
	timestampIncrement += (2.0*sink.timestampFrequency*uint(tv.Tv_usec) + 1000000.0) / 2000000

	// Then add this to our 'timestamp base':
	if sink.nextTimestampHasBeenPreset {
		// Make the returned timestamp the same as the current "fTimestampBase",
		// so that timestamps begin with the value that was previously preset:
		sink.timestampBase -= timestampIncrement
		sink.nextTimestampHasBeenPreset = false
	}

	// return RTP Timestamp
	return sink.timestampBase + timestampIncrement
}

func (sink *RTPSink) NextTimestampHasBeenPreset() bool {
	return sink.nextTimestampHasBeenPreset
}

func (sink *RTPSink) EnableRTCPReports() bool {
	return sink.enableRTCPReports
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
	timeCreated           utils.Timeval
	timeReceived          utils.Timeval
}
