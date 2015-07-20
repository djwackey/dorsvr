package liveMedia

import (
        "fmt"
)

type IFramedSource interface {
    doGetNextFrame()
    getNextFrame(buffTo []byte, maxSize uint, afterGettingFunc interface{})
}

type FramedSource struct {
    source IFramedSource
	buffTo []byte
    maxSize uint
}

func (this *FramedSource) InitFramedSource(source IFramedSource) {
    this.source = source
    fmt.Println("InitFramedSource", this.source)
}

func (this *FramedSource) getNextFrame(buffTo []byte, maxSize uint, afterGettingFunc interface{}) {
    this.maxSize = maxSize
	this.buffTo = make([]byte, maxSize)
    this.buffTo = buffTo

    fmt.Println("[FramedSource] getNextFrame", this.source)

    //this.source.doGetNextFrame()
}
