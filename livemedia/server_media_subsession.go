package livemedia

import (
	"fmt"
	"net"

	gs "github.com/djwackey/dorsvr/groupsock"
)

type IServerMediaSubSession interface {
	createNewStreamSource() IFramedSource
	createNewRTPSink(rtpGroupSock *gs.GroupSock, rtpPayloadType uint) IRTPSink
	GetStreamParameters(tcpSocketNum net.Conn, destAddr, clientSessionID string,
		clientRTPPort, clientRTCPPort, rtpChannelID, rtcpChannelID uint) *StreamParameter
	TestScaleFactor(float32) float32
	//Duration() float32
	IncrTrackNumber()
	TrackID() string
	SDPLines() string
	CNAME() string
	StartStream(clientSessionID uint, streamState *StreamState) (uint, uint)
	PauseStream(streamState *StreamState)
	SeekStream()
	DeleteStream(streamState *StreamState)
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

func (s *ServerMediaSubSession) TestScaleFactor(scale float32) float32 {
	// default implementation: Support scale = 1 only
	scale = 1.0
	return scale
}
