package liveMedia

import (
	"fmt"
	. "groupsock"
)

type BasicUDPSource struct {
	FramedSource
	inputSocket        *GroupSock
	haveStartedReading bool
}

func NewBasicUDPSource(inputSocket *GroupSock) *BasicUDPSource {
	source := new(BasicUDPSource)
	source.inputSocket = inputSocket
	source.InitFramedSource(source)
	return source
}

func (this *BasicUDPSource) doGetNextFrame() {
	if !this.haveStartedReading {
		this.incomingPacketHandler()
		this.haveStartedReading = true
	}
}

func (this *BasicUDPSource) doStopGettingFrames() {
	this.haveStartedReading = false
}

func (this *BasicUDPSource) incomingPacketHandler() {
	numBytes, err := this.inputSocket.HandleRead(this.buffTo, this.maxSize)
	if err != nil {
		fmt.Println("yanfei", err.Error())
		return
	}

	this.frameSize = uint(numBytes)

	this.afterGetting()
}
