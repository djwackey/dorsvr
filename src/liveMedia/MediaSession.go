package liveMedia

import (
	. "groupsock"
)

type MediaSession struct {
	controlPath string
}

func NewMediaSession(sdpDesc string) *MediaSession {
	mediaSession := new(MediaSession)
	mediaSession.InitWithSDP(sdpDesc)
	return mediaSession
}

type MediaSubSession struct {
	rtpSocket  *GroupSock
	rtcpSocket *GroupSock
    Sink *MediaSink
	//rtpSource RTPSource
    rtcpInstance *RTCPInstance
	controlPath string
    absStartTime string
    absEndTime string
}

func NewMediaSubSession() *MediaSubSession {
	subsession := new(MediaSubSession)
	return subsession
}

func (this *MediaSession) InitWithSDP(sdpDesc string) {
	//subsession := NewMediaSubSession()
}

func (this *MediaSession) ControlPath() string {
	return this.controlPath
}

func (this *MediaSession) absStartTime() string {
    return this.absStartTime
}

func (this *MediaSession) absEndTime() string {
    return this.absEndTime
}

// MediaSubSession Implementation
func (this *MediaSubSession) Initiate() bool {
    this.rtpSocket = NewGroupSock()
    this.rtcpSocket = NewGroupSock()
    this.rtcpInstance = NewRTCPInstance()
	return true
}

func (this *MediaSubSession) deInitiate() {
}

func (this *MediaSubSession) ControlPath() string {
	return this.controlPath
}

func (this *MediaSubSession) RtcpInstance() *RTCPInstance {
    return this.rtcpInstance
}
