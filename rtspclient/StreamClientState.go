package rtspclient

import "github.com/djwackey/dorsvr/livemedia"

type StreamClientState struct {
	Session    *livemedia.MediaSession
	Subsession *livemedia.MediaSubSession
}

func NewStreamClientState() *StreamClientState {
	return new(StreamClientState)
}

func (this *StreamClientState) Next() *livemedia.MediaSubSession {
	return this.Session.SubSession()
}
