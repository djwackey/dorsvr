package livemedia

import (
	"fmt"
	"net"

	gs "github.com/djwackey/dorsvr/groupsock"
)

type IServerMediaSubsession interface {
	getAuxSDPLine(rtpSink IMediaSink, inputSource IFramedSource) string
	setParentSession(parentSession *ServerMediaSession)
	createNewStreamSource() IFramedSource
	createNewRTPSink(rtpGroupSock *gs.GroupSock, rtpPayloadType uint) IMediaSink
	GetStreamParameters(tcpSocketNum net.Conn, destAddr, clientSessionID string,
		clientRTPPort, clientRTCPPort, rtpChannelID, rtcpChannelID uint) *StreamParameter
	TestScaleFactor(scale float32) float32
	//Duration() float32
	IncrTrackNumber()
	TrackID() string
	SDPLines() string
	CNAME() string
	StartStream(clientSessionID string, streamState *StreamState,
		rtcpRRHandler, serverRequestAlternativeByteHandler interface{}) (uint32, uint32)
	PauseStream(streamState *StreamState)
	DeleteStream(sessionID string, streamState *StreamState)
	SeekStream(sessionID string, streamState *StreamState, streamDuration float32)
}

type ServerMediaSubsession struct {
	trackNumber   uint
	trackID       string
	parentSession *ServerMediaSession
	isubsession   IServerMediaSubsession
}

func (s *ServerMediaSubsession) initBaseClass(isubsession IServerMediaSubsession) {
	s.isubsession = isubsession
}

func (s *ServerMediaSubsession) setParentSession(parentSession *ServerMediaSession) {
	s.parentSession = parentSession
}

func (s *ServerMediaSubsession) TrackID() string {
	if s.trackID == "" {
		s.trackID = fmt.Sprintf("track%d", s.trackNumber)
	}
	return s.trackID
}

func (s *ServerMediaSubsession) TrackNumber() uint {
	return s.trackNumber
}

func (s *ServerMediaSubsession) IncrTrackNumber() {
	s.trackNumber++
}

func (s *ServerMediaSubsession) getAbsoluteTimeRange(absStartTime, absEndTime *string) {
	//absStartTime = nil
	//absEndTime = nil
}

func (s *ServerMediaSubsession) rangeSDPLine() string {
	var absStart, absEnd *string
	s.getAbsoluteTimeRange(absStart, absEnd)
	if absStart != nil {
	}

	if s.parentSession == nil {
		return ""
	}

	if s.parentSession.Duration() >= 0.0 {
		return ""
	}

	ourDuration := s.Duration()
	if ourDuration == 0.0 {
		return "a=range:npt=0-\r\n"
	} else {
		return fmt.Sprintf("a=range:npt=0-%.3f\r\n", ourDuration)
	}
}

// default implementation: Support scale = 1 only
func (s *ServerMediaSubsession) TestScaleFactor(scale float32) float32 {
	scale = 1.0
	return scale
}

// default implementation: assume an unbounded session
func (s *ServerMediaSubsession) Duration() float32 {
	return 0.0
}
