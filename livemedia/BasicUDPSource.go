package livemedia

import (
	"fmt"
	gs "github.com/djwackey/dorsvr/groupsock"
)

type BasicUDPSource struct {
	FramedSource
	inputSocket        *gs.GroupSock
	haveStartedReading bool
}

func NewBasicUDPSource(inputSocket *gs.GroupSock) *BasicUDPSource {
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
