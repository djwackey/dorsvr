package liveMedia

import (
	"fmt"
	. "include"
)

type TimeCode struct {
	days     uint
	hours    uint
	minutes  uint
	seconds  uint
	pictures uint
}

func NewTimeCode() *TimeCode {
	return new(TimeCode)
}

type MPEGVideoStreamFramer struct {
	FramedFilter
	frameRate            uint
	tcSecsBase           uint
	pictureCount         uint
	pictureTimeBase      uint
	picturesAdjustment   uint
	pictureEndMarker     bool
	curGOPTimeCode       TimeCode
	preGOPTimeCode       TimeCode
	presentationTimeBase Timeval
	parser               *H264VideoStreamParser
}

func (this *MPEGVideoStreamFramer) InitMPEGVideoStreamFramer(parser *H264VideoStreamParser) {
	this.parser = parser
	this.reset()
}

func (this *MPEGVideoStreamFramer) reset() {
	GetTimeOfDay(&this.presentationTimeBase)
}

func (this *MPEGVideoStreamFramer) computePresentationTime(numAdditionalPictures uint) {
	// Computes "fPresentationTime" from the most recent GOP's
	// time_code, along with the "numAdditionalPictures" parameter:
	tc := this.curGOPTimeCode

	var pictureTime uint
	tcSecs := (((tc.days*24)+tc.hours)*60+tc.minutes)*60 + tc.seconds - this.tcSecsBase
	if this.frameRate == 0.0 {
		pictureTime = 0.0
	} else {
		pictureTime = (tc.pictures + this.picturesAdjustment + numAdditionalPictures) / this.frameRate
	}

	for pictureTime < this.pictureTimeBase { // "if" should be enough, but just in case
		if tcSecs > 0 {
			tcSecs -= 1
		}
		pictureTime += 1.0
	}
	pictureTime -= this.pictureTimeBase
	if pictureTime < 0.0 {
		pictureTime = 0.0 // sanity check
	}
	pictureSeconds := pictureTime
	pictureFractionOfSecond := pictureTime - pictureSeconds

	this.presentationTime = this.presentationTimeBase
	this.presentationTime.Tv_sec += int64(tcSecs + pictureSeconds)
	this.presentationTime.Tv_usec += int64(pictureFractionOfSecond * 1000000.0)
	if this.presentationTime.Tv_usec >= 1000000 {
		this.presentationTime.Tv_usec -= 1000000
		this.presentationTime.Tv_sec++
	}
}

func (this *MPEGVideoStreamFramer) doGetNextFrame() {
	fmt.Println(fmt.Sprintf("MPEGVideoStreamFramer::doGetNextFrame -> %p", this.source))
	this.parser.registerReadInterest(this.buffTo, this.maxSize)
	this.continueReadProcessing()
}

func (this *MPEGVideoStreamFramer) continueReadProcessing() {
	acquiredFrameSize := this.parser.parse()
	if acquiredFrameSize > 0 {
		// We were able to acquire a frame from the input.
		// It has already been copied to the reader's space.
		this.frameSize = acquiredFrameSize
		this.numTruncatedBytes = this.parser.NumTruncatedBytes()

		// "presentationTime" should have already been computed.

		// Compute "durationInMicroseconds" now:
		if this.frameRate == 0.0 || this.pictureCount < 0 {
			this.durationInMicroseconds = 0
		} else {
			this.durationInMicroseconds = this.pictureCount * 1000000 / this.frameRate
		}
		this.pictureCount = 0

		// Call our own 'after getting' function.  Because we're not a 'leaf'
		// source, we can call this directly, without risking infinite recursion.
		this.afterGetting()
	} else {
		// We were unable to parse a complete frame from the input, because:
		// - we had to read more data from the source stream, or
		// - the source stream has ended.
	}
}
