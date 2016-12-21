package rtspserver

import (
	"net"
)

type RTPInterface struct {
	gs                         *GroupSock
	owner                      interface{}
	auxReadHandlerFunc         interface{}
	tcpStreams                 *TCPStreamRecord
	nextTCPReadStreamSocketNum int
}

func NewRTPInterface(owner interface{}, gs *GroupSock) *RTPInterface {
	rtpInterface := new(RTPInterface)
	rtpInterface.gs = gs
	rtpInterface.owner = owner
	return rtpInterface
}

func (this *RTPInterface) startNetworkReading(handlerProc interface{}) {
	go handlerProc.(func())()
}

func (this *RTPInterface) stopNetworkReading() {
}

func (this *RTPInterface) GS() *GroupSock {
	return this.gs
}

func (this *RTPInterface) addStreamSocket(sockNum net.Conn, streamChannelID uint) {
	if sockNum == nil {
		return
	}

	this.tcpStreams = NewTCPStreamRecord(sockNum, streamChannelID)
}

func (this *RTPInterface) delStreamSocket() {
}

func (this *RTPInterface) sendPacket(packet []byte, packetSize uint) bool {
	return this.gs.Output(packet, packetSize, this.gs.TTL())
}

func (this *RTPInterface) handleRead(buffer []byte) (int, error) {
	readBytes, err := this.gs.HandleRead(buffer)
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
