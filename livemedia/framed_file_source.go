package livemedia

import "os"

type FramedFileSource struct {
	FramedSource
	fid *os.File
}

func (s *FramedFileSource) initFramedFileSource(source IFramedSource) {
	s.initFramedSource(source)
}
