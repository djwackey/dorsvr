package livemedia

import (
	gs "github.com/djwackey/dorsvr/groupsock"
	"net"
)

type RTPInterface struct {
	gs                         *gs.GroupSock
	owner                      interface{}
	auxReadHandlerFunc         interface{}
	nextTCPReadStreamSocketNum int
	socketDescriptors          map[net.Conn]*SocketDescriptor
	tcpStreams                 *tcpStreamRecord
}

func newRTPInterface(owner interface{}, gs *gs.GroupSock) *RTPInterface {
	return &RTPInterface{
		gs:                gs,
		owner:             owner,
		socketDescriptors: make(map[net.Conn]*SocketDescriptor),
	}
}

func (i *RTPInterface) startNetworkReading(handlerProc interface{}) {
	go handlerProc.(func())()
}

func (i *RTPInterface) stopNetworkReading() {
	i.gs.Close()
}

func (i *RTPInterface) setServerRequestAlternativeByteHandler(socketNum net.Conn, handler interface{}) {
	descriptor := i.lookupSocketDescriptor(socketNum)
	if descriptor != nil {
		descriptor.setServerRequestAlternativeByteHandler(handler)
	}
}

func (i *RTPInterface) addStreamSocket(socketNum net.Conn, streamChannelID uint) {
	if socketNum == nil {
		return
	}

	var streams *tcpStreamRecord
	for streams = i.tcpStreams; streams != nil; streams = streams.next {
		if streams.streamSocketNum == socketNum && streams.streamChannelID == streamChannelID {
			return
		}
	}

	i.tcpStreams = newTCPStreamRecord(socketNum, streamChannelID, i.tcpStreams)

	// Also, make sure this new socket is set up for receiving RTP/RTCP over TCP:
	descriptor := i.lookupSocketDescriptor(socketNum)
	descriptor.registerRTPInterface(streamChannelID, i)
}

func (i *RTPInterface) delStreamSocket(socketNum net.Conn, streamChannelID uint) {
	var streams *tcpStreamRecord
	for streams = i.tcpStreams; streams != nil; streams = streams.next {
		if streams.streamSocketNum == socketNum && streams.streamChannelID == streamChannelID {
			i.deregisterSocket(socketNum, streamChannelID)

			next := streams.next
			streams.next = nil
			streams = next
		}
	}
}

func (i *RTPInterface) sendPacket(packet []byte, packetSize uint) bool {
	return i.gs.Output(packet, packetSize, i.gs.TTL())
}

func (i *RTPInterface) handleRead(buffer []byte) (int, error) {
	readBytes, err := i.gs.HandleRead(buffer)
	return readBytes, err
}

func (i *RTPInterface) lookupSocketDescriptor(socketNum net.Conn) *SocketDescriptor {
	var existed bool
	var descriptor *SocketDescriptor
	if descriptor, existed = i.socketDescriptors[socketNum]; existed {
		return descriptor
	}

	descriptor = newSocketDescriptor(socketNum)
	i.socketDescriptors[socketNum] = descriptor
	return descriptor
}

func (i *RTPInterface) deregisterSocket(socketNum net.Conn, streamChannelID uint) {
	descriptor := i.lookupSocketDescriptor(socketNum)
	if descriptor != nil {
		i.removeSocketDescriptor(socketNum)
		descriptor.deregisterRTPInterface(streamChannelID)
	}
}

func (i *RTPInterface) removeSocketDescriptor(socketNum net.Conn) {
	delete(i.socketDescriptors, socketNum)
}

type tcpStreamRecord struct {
	streamChannelID uint
	streamSocketNum net.Conn
	next            *tcpStreamRecord
}

func newTCPStreamRecord(streamSocketNum net.Conn, streamChannelID uint, next *tcpStreamRecord) *tcpStreamRecord {
	record := new(tcpStreamRecord)
	record.streamChannelID = streamChannelID
	record.streamSocketNum = streamSocketNum
	record.next = next
	return record
}

///////////// Help Functions ///////////////
func sendRTPOverTCP(socketNum net.Conn, packet []byte, packetSize, streamChannelID int) {
	dollar := []byte{'$'}
	channelID := []byte{byte(streamChannelID)}
	socketNum.Write(dollar)
	socketNum.Write(channelID)
}

const (
	AWAITING_DOLLAR = iota
	AWAITING_STREAM_CHANNEL_ID
	AWAITING_SIZE1
	AWAITING_SIZE2
	AWAITING_PACKET_DATA
)

type SocketDescriptor struct {
	tcpReadingState                     int
	socketNum                           net.Conn
	serverRequestAlternativeByteHandler interface{}
}

func newSocketDescriptor(socketNum net.Conn) *SocketDescriptor {
	descriptor := new(SocketDescriptor)
	descriptor.socketNum = socketNum
	descriptor.tcpReadingState = AWAITING_DOLLAR
	return descriptor
}

func (s *SocketDescriptor) registerRTPInterface(streamChannelID uint, rtpInterface *RTPInterface) {
	go s.tcpReadHandler(rtpInterface)
}

func (s *SocketDescriptor) deregisterRTPInterface(streamChannelID uint) {
	s.socketNum.Close()
	if s.serverRequestAlternativeByteHandler != nil {
		s.serverRequestAlternativeByteHandler.(func(requestByte uint))(0xFE)
	}
}

func (s *SocketDescriptor) tcpReadHandler(rtpInterface *RTPInterface) {
	defer s.socketNum.Close()
	for {
		buffer := make([]byte, 1)
		if s.tcpReadingState != AWAITING_PACKET_DATA {
			_, err := gs.ReadSocket(s.socketNum, buffer)
			if err != nil {
				if s.serverRequestAlternativeByteHandler != nil {
					s.serverRequestAlternativeByteHandler.(func(requestByte uint))(0xFF)
				}
				rtpInterface.removeSocketDescriptor(s.socketNum)
				break
			}
		}
	}
}

func (s *SocketDescriptor) setServerRequestAlternativeByteHandler(handler interface{}) {
	s.serverRequestAlternativeByteHandler = handler
}
