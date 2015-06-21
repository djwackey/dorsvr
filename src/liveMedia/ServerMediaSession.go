package liveMedia

type ServerMediaSession struct {
	streamName string
}

func NewServerMediaSession() *ServerMediaSession {
	return &ServerMediaSession{}
}

func (this *ServerMediaSession) GenerateSDPDescription() string {
	return ""
}

func (this *ServerMediaSession) StreamName() string {
	return this.streamName
}

func (this *ServerMediaSession) AddSubSession(subSession ServerMediaSubSession) {
}
