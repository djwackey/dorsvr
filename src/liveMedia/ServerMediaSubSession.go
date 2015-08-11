package liveMedia

import (
	"fmt"
	. "groupsock"
	"net"
)

type IServerMediaSubSession interface {
	createNewStreamSource() IFramedSource
	createNewRTPSink(rtpGroupSock *GroupSock, rtpPayloadType uint) IRTPSink
	getStreamParameters(tcpSocketNum *net.Conn, clientRTPPort, clientRTCPPort, rtpChannelId, rtcpChannelId int) *StreamParameter
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

func (this *ServerMediaSubSession) rangeSDPLine() string {
	return "a=range:npt=0-\r\n"
}

func (this *ServerMediaSubSession) testScaleFactor(scale *float32) float32 {
    // default implementation: Support scale = 1 only
    scale = 1
    return scale
}
