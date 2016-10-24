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
	go this.incomingPacketHandler()
}

func (this *BasicUDPSource) doStopGettingFrames() {
	this.haveStartedReading = false
}

func (this *BasicUDPSource) incomingPacketHandler() {
	for {
		numBytes, err := this.inputSocket.HandleRead(this.buffTo, this.maxSize)
		if err != nil {
			fmt.Println("Failed to read from input socket.", err.Error())
			break
		}

		this.frameSize = uint(numBytes)

		this.afterGetting()
	}
}
