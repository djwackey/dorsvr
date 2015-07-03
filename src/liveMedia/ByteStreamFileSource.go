package liveMedia

import (
	"fmt"
	"os"
)

type ByteStreamFileSource struct {
    FramedFileSource
	fileSize int64
    buff     []byte
}

func NewByteStreamFileSource(fileName string) *ByteStreamFileSource {
	fid, err := os.Open(fileName)
	if err != nil {
		return nil
	}

    fileSource := new(ByteStreamFileSource)
    fileSource.fid = fid
	fileSource.buff := make([]byte, 10000)

	stat, _ := fid.Stat()
	fileSource.fileSize := stat.Size()
	return fileSource
}

func (this *ByteStreamFileSource) DoReadFromFile() {
	defer this.fid.Close()
	for {
		readBytes, err := this.fid.Read(this.buff)
		if err != nil {
			fmt.Println(err)
			break
		}

		//sp.Parse(buff)

		//fmt.Println(buff[:5])
		//nul := buff[5] & 0x1f
		fmt.Println(readBytes)
		//fmt.Println(nul)
	}
}

func (this *ByteStreamFileSource) FileSize() int64 {
	return this.fileSize
}
