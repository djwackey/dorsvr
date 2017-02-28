package rtspserver

import "github.com/djwackey/dorsvr/livemedia"

type StreamServerState struct {
	subsession  livemedia.IServerMediaSubSession
	streamToken *livemedia.StreamState
}
