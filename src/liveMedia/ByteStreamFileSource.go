package liveMedia

import (
	"fmt"
	. "include"
	"os"
)

type ByteStreamFileSource struct {
	FramedFileSource
	presentationTime Timeval
	fileSize         int64
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

func (this *ByteStreamFileSource) getNextFrame(buffTo []byte, maxSize uint, afterGettingFunc interface{}) {
	this.maxSize = maxSize
	//this.buffTo = buffTo
	fmt.Println("BSFS", afterGettingFunc, this.maxSize)

	if this.doReadFromFile() {
		afterGettingFunc.(func())()
	}
}

func (this *ByteStreamFileSource) doReadFromFile() bool {
	//defer this.fid.Close()
	readBytes, err := this.fid.Read(this.buffTo)
	if err != nil {
		fmt.Println(err)
		return false
	}

	fmt.Println(readBytes)
	//fmt.Println(this.buffTo)
	GetTimeOfDay(&this.presentationTime)
	return true
}

func (this *ByteStreamFileSource) FileSize() int64 {
	return this.fileSize
}
