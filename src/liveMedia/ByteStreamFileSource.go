package liveMedia

import (
	"fmt"
	"os"
)

type ByteStreamFileSource struct {
	mFid      *os.File
	mFileSize int64
	mBuff     []byte
}

func NewByteStreamFileSource(fileName string) *ByteStreamFileSource {
	fid, err := os.Open(fileName)
	if err != nil {
		return nil
	}

	// in
	buff := make([]byte, 10000)

	stat, _ := fid.Stat()
	fileSize := stat.Size()
	return &ByteStreamFileSource{fid, fileSize, buff}
}

func (this *ByteStreamFileSource) DoReadFromFile() {
	defer this.mFid.Close()
	for {
		readBytes, err := this.mFid.Read(this.mBuff)
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
	return this.mFileSize
}
