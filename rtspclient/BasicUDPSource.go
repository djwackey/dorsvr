package rtspclient

import (
	"fmt"
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

func (source *BasicUDPSource) doGetNextFrame() {
	go source.incomingPacketHandler()
}

func (source *BasicUDPSource) doStopGettingFrames() {
	source.haveStartedReading = false
}

func (source *BasicUDPSource) incomingPacketHandler() {
	for {
		numBytes, err := source.inputSocket.HandleRead(source.buffTo)
		if err != nil {
			fmt.Println("Failed to read from input socket.", err.Error())
			break
		}

		source.frameSize = uint(numBytes)

		source.afterGetting()
	}
}
