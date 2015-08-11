package liveMedia

import (
	"fmt"
	//"time"
	. "groupsock"
	. "include"
)

var libNameStr string = "Dor Streaming Media v"
var libVersionStr string = MEDIA_SERVER_VERSION

type ServerMediaSession struct {
	isSSM             bool
	ipAddr            string
	streamName        string
	descSDPStr        string
	infoSDPStr        string
	miscSDPLines      string
	referenceCount    int
	subsessionCounter int
	creationTime      Timeval
	subSessions       []IServerMediaSubSession
}

func NewServerMediaSession(description, streamName string) *ServerMediaSession {
	serverMediaSession := new(ServerMediaSession)
	serverMediaSession.descSDPStr = description + ", streamed by the Dor Media Server"
	serverMediaSession.infoSDPStr = streamName
	serverMediaSession.streamName = streamName
	serverMediaSession.subSessions = make([]IServerMediaSubSession, 1024)
	serverMediaSession.ipAddr, _ = OurIPAddress()

	GetTimeOfDay(&serverMediaSession.creationTime)
	return serverMediaSession
}

func (this *ServerMediaSession) GenerateSDPDescription() string {
	var sourceFilterLine string
	if this.isSSM {
		sourceFilterLine = fmt.Sprintf("a=source-filter: incl IN IP4 * %s\r\n"+
			"a=rtcp-unicast: reflection\r\n", this.ipAddr)
	} else {
		sourceFilterLine = ""
	}

	var rangeLine string
	dur := this.duration()
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
		this.creationTime.Tv_sec,
		this.creationTime.Tv_usec,
		1,
		this.ipAddr,
		this.descSDPStr,
		this.infoSDPStr,
		libNameStr, libVersionStr,
		sourceFilterLine,
		rangeLine,
		this.descSDPStr,
		this.infoSDPStr,
		this.miscSDPLines)

	// Then, add the (media-level) lines for each subsession:
	for i := 0; i < this.subsessionCounter; i++ {
		sdpLines := this.subSessions[i].SDPLines()
		sdp += sdpLines
	}

	return sdp
}

func (this *ServerMediaSession) StreamName() string {
	return this.streamName
}

func (this *ServerMediaSession) AddSubSession(subSession IServerMediaSubSession) {
	this.subSessions[this.subsessionCounter] = subSession
	this.subsessionCounter++
	subSession.IncrTrackNumber()
}

func (this *ServerMediaSession) duration() float32 {
	return 0.0
}

func (this *ServerMediaSession) testScaleFactor(scale float32) {
}
