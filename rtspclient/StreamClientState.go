package rtspclient

type StreamClientState struct {
	Session    *MediaSession
	Subsession *MediaSubSession
}

func NewStreamClientState() *StreamClientState {
	return new(StreamClientState)
}

func (this *StreamClientState) Next() *MediaSubSession {
	return this.Session.SubSession()
}
