package livemedia

import (
	"net"

	gs "github.com/djwackey/dorsvr/groupsock"
	"github.com/djwackey/gitea/log"
)

type RTPInterface struct {
	gs                         *gs.GroupSock
	owner                      interface{}
	auxReadHandlerFunc         interface{}
	nextTCPReadStreamSocketNum net.Conn
	socketDescriptors          map[net.Conn]*SocketDescriptor
	tcpStreams                 *tcpStreamRecord
	nextTCPReadSize            uint
	nextTCPReadStreamChannelID uint
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

// normal case: send as a UDP packet, also, send over each of our TCP sockets
func (i *RTPInterface) sendPacket(packet []byte, packetSize uint) bool {
	success := i.gs.Output(packet, packetSize)

	var streams *tcpStreamRecord
	for streams = i.tcpStreams; streams != nil; streams = streams.next {
		sendRTPOverTCP(streams.streamSocketNum, packet, packetSize, streams.streamChannelID)
	}

	return success
}

func (i *RTPInterface) handleRead(buffer []byte) (int, error) {
	return i.gs.HandleRead(buffer)
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
	return &tcpStreamRecord{
		next:            next,
		streamChannelID: streamChannelID,
		streamSocketNum: streamSocketNum,
	}
}

///////////// Help Functions ///////////////

// Send RTP over TCP, using the encoding defined RFC 2326, section 10.12:
func sendRTPOverTCP(socketNum net.Conn, packet []byte, packetSize, streamChannelID uint) error {
	var err error

	dollar := []byte{'$'}
	_, err = socketNum.Write(dollar)
	if err != nil {
		return err
	}

	channelID := []byte{byte(streamChannelID)}
	_, err = socketNum.Write(channelID)
	if err != nil {
		return err
	}

	netPacketSize := make([]byte, 2)
	netPacketSize[0] = byte((packetSize & 0xFF00) >> 8)
	netPacketSize[1] = byte(packetSize & 0xFF)
	_, err = socketNum.Write(netPacketSize)
	if err != nil {
		return err
	}

	_, err = socketNum.Write(packet[:packetSize])
	if err != nil {
		return err
	}

	return nil
}

const (
	awaitingDollar = iota
	awaitingStreamChannelID
	awaitingSize1
	awaitingSize2
	awaitingPacketData
)

type SocketDescriptor struct {
	tcpReadingState                     int
	streamChannelID                     uint
	sizeByte1                           uint
	socketNum                           net.Conn
	serverRequestAlternativeByteHandler interface{}
	subChannels                         map[uint]*RTPInterface
}

func newSocketDescriptor(socketNum net.Conn) *SocketDescriptor {
	return &SocketDescriptor{
		socketNum:       socketNum,
		tcpReadingState: awaitingDollar,
		subChannels:     make(map[uint]*RTPInterface),
	}
}

func (s *SocketDescriptor) registerRTPInterface(streamChannelID uint, rtpInterface *RTPInterface) {
	s.subChannels[streamChannelID] = rtpInterface
	go s.tcpReadHandler(rtpInterface)
}

func (s *SocketDescriptor) lookupRTPInterface(streamChannelID uint) (rtpInterface *RTPInterface, existed bool) {
	rtpInterface, existed = s.subChannels[streamChannelID]
	return
}

func (s *SocketDescriptor) deregisterRTPInterface(streamChannelID uint) {
	defer s.socketNum.Close()
	if s.serverRequestAlternativeByteHandler != nil {
		s.serverRequestAlternativeByteHandler.(func(requestByte uint))(0xFE)
	}
	delete(s.subChannels, streamChannelID)
}

func (s *SocketDescriptor) tcpReadHandler(rtpInterface *RTPInterface) {
	defer s.socketNum.Close()
	buffer := make([]byte, 1)
	for {
		if s.tcpReadingState != awaitingPacketData {
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

	switch s.tcpReadingState {
	case awaitingDollar:
		if buffer[0] == '$' {
			log.Debug("[SocketDescriptor::tcpReadHandler] Saw '$'")
			s.tcpReadingState = awaitingStreamChannelID
		} else {
			// This character is part of a RTSP request or command, which is handled separately:
			if s.serverRequestAlternativeByteHandler != nil && buffer[0] != 0xFF && buffer[0] != 0xFE {
				s.serverRequestAlternativeByteHandler.(func(requestByte uint))(uint(buffer[0]))
			}
		}
	case awaitingStreamChannelID:
		// The byte that we read is the stream channel id.
		if _, existed := s.lookupRTPInterface(uint(buffer[0])); existed {
			s.streamChannelID = uint(buffer[0])
			s.tcpReadingState = awaitingSize1
		} else {
			s.tcpReadingState = awaitingDollar
		}
	case awaitingSize1:
		// The byte that we read is the first (high) byte of the 16-bit RTP or RTCP packet 'size'.
		s.sizeByte1 = uint(buffer[0])
		s.tcpReadingState = awaitingSize2
	case awaitingSize2:
		// The byte that we read is the second (low) byte of the 16-bit RTP or RTCP packet 'size'.
		size := (s.sizeByte1 << 8) | uint(buffer[0])

		// Record the information about the packet data that will be read next:
		rtpInterface, existed := s.lookupRTPInterface(s.streamChannelID)
		if existed {
			rtpInterface.nextTCPReadSize = size
			rtpInterface.nextTCPReadStreamSocketNum = s.socketNum
			rtpInterface.nextTCPReadStreamChannelID = s.streamChannelID
		}
		s.tcpReadingState = awaitingPacketData
	case awaitingPacketData:
		s.tcpReadingState = awaitingDollar
	}
}

func (s *SocketDescriptor) setServerRequestAlternativeByteHandler(handler interface{}) {
	s.serverRequestAlternativeByteHandler = handler
}
