package liveMedia

import (
	//. "groupsock"
	"net"
)

const (
	AWAITING_DOLLAR = iota
	AWAITING_STREAM_CHANNEL_ID
	AWAITING_SIZE1
	AWAITING_SIZE2
	AWAITING_PACKET_DATA
)

type SocketDescriptor struct {
	tcpReadingState int
}

func NewSocketDescriptor(socketNum *net.Conn) *SocketDescriptor {
	socketDescriptor := new(SocketDescriptor)
	socketDescriptor.tcpReadingState = AWAITING_DOLLAR
	return socketDescriptor
}

func (this *SocketDescriptor) registerRTPInterface() {
	go this.tcpReadHandler()
}

func (this *SocketDescriptor) tcpReadHandler() {
	var c byte
	if this.tcpReadingState != AWAITING_PACKET_DATA {
		//readSocket()
	}

	switch this.tcpReadingState {
	case AWAITING_DOLLAR:
		if c == '$' {
			this.tcpReadingState = AWAITING_STREAM_CHANNEL_ID
		}
	case AWAITING_STREAM_CHANNEL_ID:
	case AWAITING_SIZE1:
	case AWAITING_SIZE2:
	case AWAITING_PACKET_DATA:
	}
}
