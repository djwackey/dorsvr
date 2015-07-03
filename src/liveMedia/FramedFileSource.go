package liveMedia

import "os"

type FramedFileSource struct {
    FramedSource
	fid *os.File
}
