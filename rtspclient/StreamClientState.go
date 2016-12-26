package rtspclient

import "github.com/djwackey/dorsvr/livemedia"

type StreamClientState struct {
	Session    *livemedia.MediaSession
	Subsession *livemedia.MediaSubSession
}

func NewStreamClientState() *StreamClientState {
	return new(StreamClientState)
}

func (s *StreamClientState) Next() *livemedia.MediaSubSession {
	return s.Session.SubSession()
}
