package liveMedia

type StreamClientState struct {
	Session    *MediaSession
	Subsession *MediaSubSession
}

func NewStreamClientState() *StreamClientState {
	streamClientState := new(StreamClientState)
	return streamClientState
}
