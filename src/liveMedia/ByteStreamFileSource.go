package liveMedia

import (
	"fmt"
	"os"
    . "include"
)

type ByteStreamFileSource struct {
	FramedFileSource
    presentationTime Timeval
	fileSize int64
}

func NewByteStreamFileSource(fileName string) *ByteStreamFileSource {
	fid, err := os.Open(fileName)
	if err != nil {
        fmt.Println(err, fileName)
		return nil
	}

	fileSource := new(ByteStreamFileSource)
	fileSource.fid = fid

    fmt.Println("[ByteStreamFileSource]", fileSource)

	stat, _ := fid.Stat()
	fileSource.fileSize = stat.Size()
    fileSource.InitFramedFileSource(fileSource)
	return fileSource
}

func (this *ByteStreamFileSource) doGetNextFrame() {
	this.doReadFromFile()
}

func (this *ByteStreamFileSource) doReadFromFile() {
	//defer this.fid.Close()
    readBytes, err := this.fid.Read(this.buffTo)
    if err != nil {
	    fmt.Println(err)
		return
	}

    fmt.Println(readBytes)
	GetTimeOfDay(&this.presentationTime)
}

func (this *ByteStreamFileSource) FileSize() int64 {
	return this.fileSize
}
