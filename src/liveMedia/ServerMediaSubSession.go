package liveMedia

import (
	"fmt"
	. "groupsock"
)

type IServerMediaSubSession interface {
	CreateNewStreamSource() IFramedSource
	CreateNewRTPSink(rtpGroupSock *GroupSock, rtpPayloadType int) IRTPSink
	IncrTrackNumber()
	SDPLines() string
}

type ServerMediaSubSession struct {
	trackNumber int
	trackId     string
	isubsession IServerMediaSubSession
}

func (this *ServerMediaSubSession) InitServerMediaSubSession(isubsession IServerMediaSubSession) {
	this.isubsession = isubsession
}

func (this *OnDemandServerMediaSubSession) TrackId() string {
	if this.trackId == "" {
		this.trackId = fmt.Sprintf("track%d", this.trackNumber)
	}
	return this.trackId
}

func (this *ServerMediaSubSession) TrackNumber() int {
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
