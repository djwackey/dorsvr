package liveMedia

type RTPSource type {
    FramedSource
}

func NewRTPSource() *RTPSource {
    return new(RTPSource)
}
