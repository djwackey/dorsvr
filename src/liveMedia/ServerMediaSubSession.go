package liveMedia

type ServerMediaSubSession interface {
	StartStream()
	PauseStream()
	SeekStream()
	DeleteStream()

	//NewStreamSource(clientSessionId uint, estBitrate uint)
	//NewRTPSink()
}
