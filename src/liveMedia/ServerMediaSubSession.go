package liveMedia

import (
	"fmt"
	. "groupsock"
)

type IServerMediaSubSession interface {
	CreateNewStreamSource() IFramedSource
	CreateNewRTPSink(rtpGroupSock *GroupSock, rtpPayloadType uint) IRTPSink
	getStreamParameters(rtpChannelId, rtcpChannelId int) *StreamParameter
	IncrTrackNumber()
	SDPLines() string
	CNAME() string
	startStream(streamState *StreamState)
	pauseStream(streamState *StreamState)
	//seekStream()
	deleteStream(streamState *StreamState)
}

type ServerMediaSubSession struct {
	trackNumber uint
	trackId     string
	isubsession IServerMediaSubSession
}

func (this *ServerMediaSubSession) InitServerMediaSubSession(isubsession IServerMediaSubSession) {
	this.isubsession = isubsession
}

func (this *ServerMediaSubSession) TrackId() string {
	if this.trackId == "" {
		this.trackId = fmt.Sprintf("track%d", this.trackNumber)
	}
	return this.trackId
}

func (this *ServerMediaSubSession) TrackNumber() uint {
	return this.trackNumber
}

func (this *ServerMediaSubSession) IncrTrackNumber() {
	this.trackNumber++
}

func rangeSDPLine() string {
	return "a=range:npt=0-\r\n"
}

func getAuxSDPLine(rtpSink *RTPSink) interface{} {
	if rtpSink == nil {
		return nil
	}

	return rtpSink.AuxSDPLine()
}
