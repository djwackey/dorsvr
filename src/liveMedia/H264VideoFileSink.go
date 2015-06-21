package liveMedia

import "os"

type H264VideoFileSink struct {
    fid *os.File
}

func NewH264VideoFileSink(fileName string) *H264VideoFileSink {
    fid := OpenOutputFile(fileName)
    return &H264VideoFileSink{fid}
}
