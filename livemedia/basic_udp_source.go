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

func newBasicUDPSource(inputSocket *gs.GroupSock) *BasicUDPSource {
	source := new(BasicUDPSource)
	source.inputSocket = inputSocket
	source.initFramedSource(source)
	return source
}

func (s *BasicUDPSource) doGetNextFrame() error {
	go s.incomingPacketHandler()
	return nil
}

func (s *BasicUDPSource) doStopGettingFrames() error {
	s.haveStartedReading = false
	return nil
}

func (s *BasicUDPSource) incomingPacketHandler() {
	for {
		numBytes, err := s.inputSocket.HandleRead(s.buffTo)
		if err != nil {
			fmt.Println("Failed to read from input socket.", err.Error())
			break
		}

		s.frameSize = uint(numBytes)

		s.afterGetting()
	}
}
