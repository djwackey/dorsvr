package liveMedia

type H264VideoFileServerMediaSubsession struct {
    mFileName string
    mFileSize int64
}

func NewH264VideoFileServerMediaSubsession() *H264VideoFileServerMediaSubsession {
    return &H264VideoFileServerMediaSubsession{}
}

func (this *H264VideoFileServerMediaSubsession) CreateNewStreamSource(estBitrate uint) *H264VideoStreamFramer {
    estBitrate = 500    // kbps, estimate

    // Create the video source:
    fileSource := NewByteStreamFileSource(this.mFileName)
    if fileSource == nil {
        return nil
    }
    this.mFileSize = fileSource.FileSize()

    // Create a framer for the Video Elementary Stream:
    return NewH264VideoStreamFramer()
}

func (this *H264VideoFileServerMediaSubsession) CreateNewRTPSink() {
    //return NewH264VideoRTPSink()
}
