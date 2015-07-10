package liveMedia


type StreamClientState struct {
    MediaSession    *Session
    MediaSubSession *Subsession
}

func NewStreamClientState() *StreamClientState {
    streamClientState := new(StreamClientState)
    return streamClientState
}
