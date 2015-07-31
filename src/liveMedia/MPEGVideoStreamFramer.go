package liveMedia

type MPEGVideoStreamFramer struct {
	FramedFilter
    pictureCount uint
    frameRate float32
}

func (this *MPEGVideoStreamFramer) InitMPEGVideoStreamFramer() {
}

func (this *MPEGVideoStreamFramer) doGetNextFrame() {
    this.parser.registerReadInterest(this.buffTo, this.maxSize)
    this.continueReadProcessing()
}

func (this *MPEGVideoStreamFramer) continueReadProcessing() {
    acquiredFrameSize := this.parser.parse()
    if acquiredFrameSize > 0 {
        // We were able to acquire a frame from the input.
        // It has already been copied to the reader's space.
        this.frameSize = acquiredFrameSize
        this.numTruncatedBytes = this.parser.numTruncatedBytes()

        // "fPresentationTime" should have already been computed.

        // Compute "fDurationInMicroseconds" now:
        if this.frameRate == 0.0 || this.pictureCount < 0 {
            this.durationInMicroseconds = 0
        } else {
            this.durationInMicroseconds = this.pictureCount*1000000/this.frameRate
        }
        this.pictureCount = 0

        // Call our own 'after getting' function.  Because we're not a 'leaf'
        // source, we can call this directly, without risking infinite recursion.
        afterGetting(this)
    } else {
        // We were unable to parse a complete frame from the input, because:
        // - we had to read more data from the source stream, or
        // - the source stream has ended.
    }
}


type MPEGVideoStreamParser struct {
	StreamParser
}


func (this *MPEGVideoStreamParser) registerReadInterest(to []byte, maxSize uint) {
    fStartOfFrame = fTo
    fSavedTo = to
    fLimit = to + maxSize
    fNumTruncatedBytes = 0
    fSavedNumTruncatedBytes = 0
}
