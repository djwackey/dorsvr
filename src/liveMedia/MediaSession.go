package liveMedia

type MediaSession struct {
    controlPath string
}

func NewMediaSession(sdpDesc string) *MediaSession {
    mediaSession := new(MediaSession)
    mediaSession.InitWithSDP(sdpDesc)
    return mediaSession
}

type MediaSubSession struct {
    rtpSocket *GroupSock
    rtcpSocket *GroupSock
    rtpSource RTPSource

}

func NewMediaSubSession() *MediaSubSession {
    subsession := new(MediaSubSession)
    return subsession
}

func (this *MediaSession) InitWithSDP(sdpDesc string) {
    //subsession := NewMediaSubSession()
}

func (this *MediaSession) ControlPath() string {
    this.controlPath
}

func (this *MediaSubSession) Initiate() bool {
    return true
}
