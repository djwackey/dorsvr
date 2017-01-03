package livemedia

import (
	"fmt"
	"net"

	gs "github.com/djwackey/dorsvr/groupsock"
)

type IServerMediaSubSession interface {
	createNewStreamSource() IFramedSource
	createNewRTPSink(rtpGroupSock *gs.GroupSock, rtpPayloadType uint) IRTPSink
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

func (s *ServerMediaSubSession) InitServerMediaSubSession(isubsession IServerMediaSubSession) {
	s.isubsession = isubsession
}

func (s *ServerMediaSubSession) TrackID() string {
	if s.trackID == "" {
		s.trackID = fmt.Sprintf("track%d", s.trackNumber)
	}
	return s.trackID
}

func (s *ServerMediaSubSession) TrackNumber() uint {
	return s.trackNumber
}

func (s *ServerMediaSubSession) IncrTrackNumber() {
	s.trackNumber++
}

func (s *ServerMediaSubSession) rangeSDPLine() string {
	return "a=range:npt=0-\r\n"
}

func (s *ServerMediaSubSession) testScaleFactor(scale float32) float32 {
	// default implementation: Support scale = 1 only
	scale = 1.0
	return scale
}
