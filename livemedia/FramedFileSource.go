package livemedia

import "os"

type FramedFileSource struct {
	FramedSource
	fid *os.File
}

func (s *FramedFileSource) InitFramedFileSource(source IFramedSource) {
	s.InitFramedSource(source)
}
