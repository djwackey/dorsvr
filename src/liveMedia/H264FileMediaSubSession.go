package liveMedia

type H264FileMediaSubSession struct {
	mFileName string
	mFileSize int64
}

func NewH264FileMediaSubSession() *H264FileMediaSubSession {
	return &H264FileMediaSubSession{}
}

func (this *H264FileMediaSubSession) CreateNewStreamSource(estBitrate uint) *H264VideoStreamFramer {
	estBitrate = 500 // kbps, estimate

	// Create the video source:
	fileSource := NewByteStreamFileSource(this.mFileName)
	if fileSource == nil {
		return nil
	}
	this.mFileSize = fileSource.FileSize()

	// Create a framer for the Video Elementary Stream:
	return NewH264VideoStreamFramer()
}

func (this *H264FileMediaSubSession) CreateNewRTPSink() {
	//return NewH264VideoRTPSink(this.mFileName)
}
