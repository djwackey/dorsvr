package livemedia

import (
	//"github.com/djwackey/dorsvr/log"
	sys "syscall"
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
	presentationTimeBase sys.Timeval
	parser               *H264VideoStreamParser
}

func (f *MPEGVideoStreamFramer) initMPEGVideoStreamFramer(parser *H264VideoStreamParser) {
	f.parser = parser
	f.reset()
}

func (f *MPEGVideoStreamFramer) reset() {
	sys.Gettimeofday(&f.presentationTimeBase)
}

// Computes "presentationTime" from the most recent GOP's
// time_code, along with the "numAdditionalPictures" parameter:
func (f *MPEGVideoStreamFramer) computePresentationTime(numAdditionalPictures uint) {
	tc := f.curGOPTimeCode

	var pictureTime uint
	tcSecs := (((tc.days*24)+tc.hours)*60+tc.minutes)*60 + tc.seconds - f.tcSecsBase
	if f.frameRate == 0.0 {
		pictureTime = 0.0
	} else {
		pictureTime = (tc.pictures + f.picturesAdjustment + numAdditionalPictures) / f.frameRate
	}

	for pictureTime < f.pictureTimeBase { // "if" should be enough, but just in case
		if tcSecs > 0 {
			tcSecs -= 1
		}
		pictureTime += 1.0
	}
	pictureTime -= f.pictureTimeBase
	if pictureTime < 0.0 {
		pictureTime = 0.0 // sanity check
	}
	pictureSeconds := pictureTime
	pictureFractionOfSecond := pictureTime - pictureSeconds

	f.presentationTime = f.presentationTimeBase
	f.presentationTime.Sec += int32(tcSecs + pictureSeconds)
	f.presentationTime.Usec += int32(pictureFractionOfSecond * 1000000.0)
	if f.presentationTime.Usec >= 1000000 {
		f.presentationTime.Usec -= 1000000
		f.presentationTime.Sec++
	}
}

func (f *MPEGVideoStreamFramer) doGetNextFrame() error {
	f.parser.registerReadInterest(f.buffTo, f.maxSize)
	f.continueReadProcessing()
	return nil
}

func (f *MPEGVideoStreamFramer) continueReadProcessing() {
	acquiredFrameSize, err := f.parser.parse()
	if err == nil {
		// We were able to acquire a frame from the input.
		// It has already been copied to the reader's space.
		f.frameSize = acquiredFrameSize
		f.numTruncatedBytes = f.parser.numTruncatedBytes

		// "presentationTime" should have already been computed.

		// Compute "durationInMicroseconds" now:
		if f.frameRate == 0.0 || f.pictureCount < 0 {
			f.durationInMicroseconds = 0
		} else {
			f.durationInMicroseconds = f.pictureCount * 1000000 / f.frameRate
		}
		f.pictureCount = 0

		// Call our own 'after getting' function.  Because we're not a 'leaf'
		// source, we can call this directly, without risking infinite recursion.
		f.afterGetting()
	} else {
		if err.Error() == "EOF" {
			// We were unable to parse a complete frame from the input, because:
			// - we had to read more data from the source stream, or
			// - the source stream has ended.
		} else {
			f.continueReadProcessing()
		}
	}
}
