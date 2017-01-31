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
	tcpStreams                 *TCPStreamRecord
}

func NewRTPInterface(owner interface{}, gs *gs.GroupSock) *RTPInterface {
	rtpInterface := new(RTPInterface)
	rtpInterface.gs = gs
	rtpInterface.owner = owner
	return rtpInterface
}

func (i *RTPInterface) startNetworkReading(handlerProc interface{}) {
	go handlerProc.(func())()
}

func (i *RTPInterface) stopNetworkReading() {
}

func (i *RTPInterface) GS() *gs.GroupSock {
	return i.gs
}

func (i *RTPInterface) addStreamSocket(sockNum net.Conn, streamChannelID uint) {
	if sockNum == nil {
		return
	}

	i.tcpStreams = NewTCPStreamRecord(sockNum, streamChannelID)
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

type TCPStreamRecord struct {
	streamChannelID uint
	streamSocketNum net.Conn
}

func NewTCPStreamRecord(streamSocketNum net.Conn, streamChannelID uint) *TCPStreamRecord {
	tcpStreamRecord := new(TCPStreamRecord)
	tcpStreamRecord.streamChannelID = streamChannelID
	tcpStreamRecord.streamSocketNum = streamSocketNum
	return tcpStreamRecord
}

///////////// Help Functions ///////////////
func sendRTPOverTCP(socketNum net.Conn, packet []byte, packetSize, streamChannelID int) {
	dollar := []byte{'$'}
	channelID := []byte{byte(streamChannelID)}
	socketNum.Write(dollar)
	socketNum.Write(channelID)
}
