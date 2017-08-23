package livemedia

import (
	"fmt"
	sys "syscall"

	gs "github.com/djwackey/dorsvr/groupsock"
)

var libNameStr string = "Dor Streaming Media v"
var libVersionStr string = mediaServerVersion

type ServerMediaSession struct {
	isSSM             bool
	ipAddr            string
	streamName        string
	descSDPStr        string
	infoSDPStr        string
	miscSDPLines      string
	referenceCount    int
	SubsessionCounter int
	creationTime      sys.Timeval
	Subsessions       []IServerMediaSubsession
}

func NewServerMediaSession(description, streamName string) *ServerMediaSession {
	session := new(ServerMediaSession)
	session.descSDPStr = description + ", streamed by the Dor Media Server"
	session.infoSDPStr = streamName
	session.streamName = streamName
	session.Subsessions = make([]IServerMediaSubsession, 1024)
	session.ipAddr, _ = gs.OurIPAddress()

	sys.Gettimeofday(&session.creationTime)
	return session
}

func (s *ServerMediaSession) GenerateSDPDescription() string {
	var sourceFilterLine string
	if s.isSSM {
		sourceFilterLine = fmt.Sprintf("a=source-filter: incl IN IP4 * %s\r\n"+
			"a=rtcp-unicast: reflection\r\n", s.ipAddr)
	} else {
		sourceFilterLine = ""
	}

	var rangeLine string
	dur := s.Duration()
	if dur == 0.0 {
		rangeLine = "a=range:npt=0-\r\n"
	} else if dur > 0.0 {
		rangeLine = fmt.Sprintf("a=range:npt=0-%.3f\r\n", dur)
	}

	sdpPrefixFmt := "v=0\r\n" +
		"o=- %d%06d %d IN IP4 %s\r\n" +
		"s=%s\r\n" +
		"i=%s\r\n" +
		"t=0 0\r\n" +
		"a=tool:%s%s\r\n" +
		"a=type:broadcast\r\n" +
		"a=control:*\r\n" +
		"%s" +
		"%s" +
		"a=x-qt-text-nam:%s\r\n" +
		"a=x-qt-text-inf:%s\r\n" +
		"%s"

	sdp := fmt.Sprintf(sdpPrefixFmt,
		s.creationTime.Sec,
		s.creationTime.Usec,
		1,
		s.ipAddr,
		s.descSDPStr,
		s.infoSDPStr,
		libNameStr, libVersionStr,
		sourceFilterLine,
		rangeLine,
		s.descSDPStr,
		s.infoSDPStr,
		s.miscSDPLines)

	// Then, add the (media-level) lines for each subsession:
	for i := 0; i < s.SubsessionCounter; i++ {
		sdpLines := s.Subsessions[i].SDPLines()
		sdp += sdpLines
	}

	return sdp
}

func (s *ServerMediaSession) StreamName() string {
	return s.streamName
}

func (s *ServerMediaSession) AddSubsession(subsession IServerMediaSubsession) {
	s.Subsessions[s.SubsessionCounter] = subsession
	s.SubsessionCounter++
	subsession.setParentSession(s)
	subsession.IncrTrackNumber()
}

func (s *ServerMediaSession) Duration() float32 {
	return 0.0
}

func (s *ServerMediaSession) TestScaleFactor() float32 {
	return 1.0
}
