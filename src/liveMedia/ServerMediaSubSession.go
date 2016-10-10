package liveMedia

import (
	"fmt"
	. "groupsock"
	"net"
)

type IServerMediaSubSession interface {
	createNewStreamSource() IFramedSource
	createNewRTPSink(rtpGroupSock *GroupSock, rtpPayloadType uint) IRTPSink
	getStreamParameters(tcpSocketNum net.Conn, destAddr, clientSessionID string,
		clientRTPPort, clientRTCPPort, rtpChannelID, rtcpChannelID uint) *StreamParameter
	testScaleFactor(float32) float32
	//Duration() float32
	IncrTrackNumber()
	TrackID() string
	SDPLines() string
	CNAME() string
	startStream(clientSessionID uint, streamState *StreamState) (uint, uint)
	pauseStream(streamState *StreamState)
	seekStream()
	deleteStream(streamState *StreamState)
}

type ServerMediaSubSession struct {
	trackNumber uint
	trackID     string
	isubsession IServerMediaSubSession
}

func (this *ServerMediaSubSession) InitServerMediaSubSession(isubsession IServerMediaSubSession) {
	this.isubsession = isubsession
}

func (this *ServerMediaSubSession) TrackID() string {
	if this.trackID == "" {
		this.trackID = fmt.Sprintf("track%d", this.trackNumber)
	}
	return this.trackID
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

func (this *ServerMediaSubSession) testScaleFactor(scale float32) float32 {
	// default implementation: Support scale = 1 only
	scale = 1.0
	return scale
}
