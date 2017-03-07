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
	tcpStreams                 *tcpStreamRecord
}

func newRTPInterface(owner interface{}, gs *gs.GroupSock) *RTPInterface {
	rtpInterface := new(RTPInterface)
	rtpInterface.owner = owner
	rtpInterface.gs = gs
	return rtpInterface
}

func (i *RTPInterface) startNetworkReading(handlerProc interface{}) {
	go handlerProc.(func())()
}

func (i *RTPInterface) stopNetworkReading() {
}

func (i *RTPInterface) setServerRequestAlternativeByteHandler(socketNum net.Conn, handler interface{}) {
	descriptor := lookupSocketDescriptor(socketNum)
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

	descriptor := lookupSocketDescriptor(socketNum)
	descriptor.registerRTPInterface(streamChannelID, i)
}

func (i *RTPInterface) delStreamSocket() {
}

func (i *RTPInterface) sendPacket(packet []byte, packetSize uint) bool {
	return i.gs.Output(packet, packetSize, i.gs.TTL())
}

func (i *RTPInterface) handleRead(buffer []byte) (int, error) {
	readBytes, err := i.gs.HandleRead(buffer)
	return readBytes, err
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

type SocketDescriptor struct {
	socketNum                           net.Conn
	serverRequestAlternativeByteHandler interface{}
}

func newSocketDescriptor(socketNum net.Conn) *SocketDescriptor {
	descriptor := new(SocketDescriptor)
	descriptor.socketNum = socketNum
	return descriptor
}

func (s *SocketDescriptor) registerRTPInterface(streamChannelID uint, rtpInterface *RTPInterface) {
	go s.tcpReadHandler()
}

func (s *SocketDescriptor) tcpReadHandler() {
	for {
		break
	}
}

func (s *SocketDescriptor) setServerRequestAlternativeByteHandler(handler interface{}) {
	s.serverRequestAlternativeByteHandler = handler
}

func lookupSocketDescriptor(socketNum net.Conn) *SocketDescriptor {
	return newSocketDescriptor(socketNum)
}
