package rtspclient

import "github.com/djwackey/dorsvr/livemedia"

type StreamClientState struct {
	Session    *livemedia.MediaSession
	Subsession *livemedia.MediaSubsession
}

func newStreamClientState() *StreamClientState {
	return new(StreamClientState)
}

func (s *StreamClientState) Next() *livemedia.MediaSubsession {
	return s.Session.Subsession()
}
