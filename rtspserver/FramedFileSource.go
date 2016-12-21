package rtspserver

import "os"

type FramedFileSource struct {
	FramedSource
	fid *os.File
}

func (this *FramedFileSource) InitFramedFileSource(source IFramedSource) {
	this.InitFramedSource(source)
}
