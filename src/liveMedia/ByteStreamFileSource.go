package liveMedia

import (
	"fmt"
	"os"
	"utils"
)

type ByteStreamFileSource struct {
	FramedFileSource
	presentationTime      utils.Timeval
	fileSize              int64
	numBytesToStream      int64
	lastPlayTime          uint
	playTimePerFrame      uint
	preferredFrameSize    uint
	haveStartedReading    bool
	limitNumBytesToStream bool
}

func NewByteStreamFileSource(fileName string) *ByteStreamFileSource {
	fid, err := os.Open(fileName)
	if err != nil {
		fmt.Println(err, fileName)
		return nil
	}

	fileSource := new(ByteStreamFileSource)
	fileSource.fid = fid

	fileSource.buffTo = make([]byte, 20000)

	stat, _ := fid.Stat()
	fileSource.fileSize = stat.Size()
	fileSource.InitFramedFileSource(fileSource)
	return fileSource
}

func (this *ByteStreamFileSource) doGetNextFrame() {
	if this.limitNumBytesToStream && this.numBytesToStream == 0 {
		this.handleClosure()
		return
	}

	this.doReadFromFile()
}

func (this *ByteStreamFileSource) doStopGettingFrames() {
	defer this.fid.Close()
	this.haveStartedReading = false
}

func (this *ByteStreamFileSource) doReadFromFile() bool {
	/*readBytes*/ _, err := this.fid.Read(this.buffTo)
	if err != nil {
		fmt.Println(err)
		return false
	}

	//fmt.Println(readBytes)
	//fmt.Println(this.buffTo)

	// Set the 'presentation time':
	if this.playTimePerFrame > 0 && this.preferredFrameSize > 0 {
		if this.presentationTime.Tv_sec == 0 && this.presentationTime.Tv_usec == 0 {
			// This is the first frame, so use the current time:
			utils.GetTimeOfDay(&this.presentationTime)
		} else {
			// Increment by the play time of the previous data:
			uSeconds := this.presentationTime.Tv_usec + int64(this.lastPlayTime)
			this.presentationTime.Tv_sec += uSeconds / 1000000
			this.presentationTime.Tv_usec = uSeconds % 1000000
		}

		// Remember the play time of this data:
		this.lastPlayTime = (this.playTimePerFrame * this.frameSize) / this.preferredFrameSize
		this.durationInMicroseconds = this.lastPlayTime
	} else {
		// We don't know a specific play time duration for this data,
		// so just record the current time as being the 'presentation time':
		utils.GetTimeOfDay(&this.presentationTime)
	}

	this.afterGetting()
	return true
}

func (this *ByteStreamFileSource) FileSize() int64 {
	return this.fileSize
}
