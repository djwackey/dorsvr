package livemedia

import (
	"errors"
	"fmt"
	"os"
	sys "syscall"

	"github.com/djwackey/gitea/log"
)

type ByteStreamFileSource struct {
	FramedFileSource
	presentationTime   sys.Timeval
	fileSize           int64
	lastPlayTime       uint
	playTimePerFrame   uint
	preferredFrameSize uint
}

func newByteStreamFileSource(fileName string) *ByteStreamFileSource {
	fid, err := os.Open(fileName)
	if err != nil {
		fmt.Println(err, fileName)
		return nil
	}
	stat, _ := fid.Stat()

	fileSource := new(ByteStreamFileSource)
	fileSource.fid = fid
	fileSource.fileSize = stat.Size()
	fileSource.initFramedFileSource(fileSource)
	return fileSource
}

func (s *ByteStreamFileSource) destroy() {
	s.stopGettingFrames()
}

func (s *ByteStreamFileSource) doGetNextFrame() error {
	if !s.source.isAwaitingData() {
		s.doStopGettingFrames()
		return errors.New("file source is not awaiting data.")
	}

	if err := s.doReadFromFile(); err != nil {
		return err
	}

	return nil
}

func (s *ByteStreamFileSource) doStopGettingFrames() error {
	return s.fid.Close()
}

func (s *ByteStreamFileSource) doReadFromFile() error {
	frameSize, err := s.fid.Read(s.buffTo)
	if err != nil {
		log.Trace("[ByteStreamFileSource::doReadFromFile] Failed to read bytes from file.%s", err.Error())
		s.handleClosure()
		return err
	}
	s.frameSize = uint(frameSize)

	// Set the 'presentation time':
	if s.playTimePerFrame > 0 && s.preferredFrameSize > 0 {
		if s.presentationTime.Sec == 0 && s.presentationTime.Usec == 0 {
			// This is the first frame, so use the current time:
			sys.Gettimeofday(&s.presentationTime)
		} else {
			// Increment by the play time of the previous data:
			uSeconds := s.presentationTime.Usec + int32(s.lastPlayTime)
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
	return nil
}

func (s *ByteStreamFileSource) FileSize() int64 {
	return s.fileSize
}
