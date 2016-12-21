package rtspserver

import (
	"fmt"
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
	addStreamSocket(sockNum net.Conn, streamChannelId uint)
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

func (this *RTPSink) InitRTPSink(rtpSink IRTPSink, gs *GroupSock, rtpPayloadType,
	rtpTimestampFrequency uint, rtpPayloadFormatName string) {
	this.InitMediaSink(rtpSink)
	this.rtpInterface = NewRTPInterface(this, gs)
	this.rtpPayloadType = rtpPayloadType
	this.rtpTimestampFrequency = rtpTimestampFrequency
	this.rtpPayloadFormatName = rtpPayloadFormatName
}

func (this *RTPSink) SSRC() uint {
	return this.ssrc
}

func (this *RTPSink) addStreamSocket(sockNum net.Conn, streamChannelID uint) {
	this.rtpInterface.addStreamSocket(sockNum, streamChannelID)
}

func (this *RTPSink) delStreamSocket() {
	this.rtpInterface.delStreamSocket()
}

func (this *RTPSink) currentSeqNo() uint {
	return this.seqNo
}

func (this *RTPSink) SdpMediaType() string {
	return "data"
}

func (this *RTPSink) RtpPayloadType() uint {
	return this.rtpPayloadType
}

func (this *RTPSink) RtpmapLine() string {
	var rtpmapLine string
	if this.rtpPayloadType >= 96 {
		encodingParamsPart := ""
		rtpmapFmt := "a=rtpmap:%d %s/%d%s\r\n"
		rtpmapLine = fmt.Sprintf(rtpmapFmt,
			this.RtpPayloadType(),
			this.RtpPayloadFormatName(),
			this.RtpTimestampFrequency(), encodingParamsPart)
	}

	return rtpmapLine
}

func (this *RTPSink) RtpPayloadFormatName() string {
	return this.rtpPayloadFormatName
}

func (this *RTPSink) RtpTimestampFrequency() uint {
	return this.rtpTimestampFrequency
}

func (this *RTPSink) presetNextTimestamp() uint {
	var timeNow Timeval
	GetTimeOfDay(&timeNow)

	tsNow := this.convertToRTPTimestamp(timeNow)
	this.timestampBase = tsNow
	this.nextTimestampHasBeenPreset = true

	return tsNow
}

func (this *RTPSink) convertToRTPTimestamp(tv Timeval) uint {
	// Begin by converting from "struct timeval" units to RTP timestamp units:
	timestampIncrement := this.timestampFrequency * uint(tv.Tv_sec)
	timestampIncrement += (2.0*this.timestampFrequency*uint(tv.Tv_usec) + 1000000.0) / 2000000

	// Then add this to our 'timestamp base':
	if this.nextTimestampHasBeenPreset {
		// Make the returned timestamp the same as the current "fTimestampBase",
		// so that timestamps begin with the value that was previously preset:
		this.timestampBase -= timestampIncrement
		this.nextTimestampHasBeenPreset = false
	}

	rtpTimestamp := this.timestampBase + timestampIncrement
	return rtpTimestamp
}

func (this *RTPSink) NextTimestampHasBeenPreset() bool {
	return this.nextTimestampHasBeenPreset
}

func (this *RTPSink) EnableRTCPReports() bool {
	return this.enableRTCPReports
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
	timeCreated           Timeval
	timeReceived          Timeval
}
