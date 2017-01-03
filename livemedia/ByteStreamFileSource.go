package livemedia

import (
	"fmt"
	"os"
	sys "syscall"
)

type ByteStreamFileSource struct {
	FramedFileSource
	presentationTime      sys.Timeval
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

func (s *ByteStreamFileSource) doGetNextFrame() {
	if s.limitNumBytesToStream && s.numBytesToStream == 0 {
		s.handleClosure()
		return
	}

	s.doReadFromFile()
}

func (s *ByteStreamFileSource) doStopGettingFrames() {
	defer s.fid.Close()
	s.haveStartedReading = false
}

func (s *ByteStreamFileSource) doReadFromFile() bool {
	/*readBytes*/ _, err := s.fid.Read(s.buffTo)
	if err != nil {
		fmt.Println(err)
		return false
	}

	//fmt.Println(readBytes)
	//fmt.Println(this.buffTo)

	// Set the 'presentation time':
	if s.playTimePerFrame > 0 && s.preferredFrameSize > 0 {
		if s.presentationTime.Sec == 0 && s.presentationTime.Usec == 0 {
			// This is the first frame, so use the current time:
			sys.Gettimeofday(&s.presentationTime)
		} else {
			// Increment by the play time of the previous data:
			uSeconds := s.presentationTime.Usec + int64(s.lastPlayTime)
			s.presentationTime.Sec += uSeconds / 1000000
			s.presentationTime.Usec = uSeconds % 1000000
		}

		// Remember the play time of this data:
		s.lastPlayTime = (s.playTimePerFrame * s.frameSize) / s.preferredFrameSize
		s.durationInMicroseconds = s.lastPlayTime
	} else {
		// We don't know a specific play time duration for this data,
		// so just record the current time as being the 'presentation time':
		sys.Gettimeofday(&s.presentationTime)
	}

	s.afterGetting()
	return true
}

func (s *ByteStreamFileSource) FileSize() int64 {
	return s.fileSize
}
