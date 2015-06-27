package liveMedia

type H264FileMediaSubSession struct {
	FileServerMediaSubSession
}

func NewH264FileMediaSubSession() *H264FileMediaSubSession {
	return &H264FileMediaSubSession{}
}

func (this *H264FileMediaSubSession) CreateNewStreamSource(estBitrate uint) *H264VideoStreamFramer {
	estBitrate = 500 // kbps, estimate

	// Create the video source:
	fileSource := NewByteStreamFileSource(this.fileName)
	if fileSource == nil {
		return nil
	}
	this.fileSize = fileSource.FileSize()

	// Create a framer for the Video Elementary Stream:
	return NewH264VideoStreamFramer()
}

func (this *H264FileMediaSubSession) CreateNewRTPSink() {
	//return NewH264VideoRTPSink()
}
