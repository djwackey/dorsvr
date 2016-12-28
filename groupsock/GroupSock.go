package groupsock

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Socket struct {
	socketNum net.Conn
	portNum   uint
}

func (s *Socket) Close() {
	s.socketNum.Close()
}

type OutputSocket struct {
	Socket
	sourcePort  uint
	lastSentTTL uint
}

func (o *OutputSocket) write(destAddr string, portNum uint, buffer []byte, bufferSize uint) bool {
	udpConn := SetupDatagramSocket(destAddr, portNum)
	return writeSocket(udpConn, buffer)
}

type GroupSock struct {
	OutputSocket
	dests []*destRecord
	ttl   uint
}

func NewGroupSock(addrStr string, portNum uint) *GroupSock {
	socketNum := SetupDatagramSocket(addrStr, portNum)
	if socketNum == nil {
		return nil
	}

	gs := new(GroupSock)
	gs.ttl = 255
	gs.portNum = portNum
	gs.socketNum = socketNum
	gs.AddDestination(addrStr, portNum)
	return gs
}

func (g *GroupSock) Output(buffer []byte, bufferSize, ttlToSend uint) bool {
	var writeSuccess bool
	for i := 0; i < len(g.dests); i++ {
		dest := g.dests[i]
		if g.write(dest.addrStr, dest.portNum, buffer, bufferSize) {
			writeSuccess = true
		}
	}
	return writeSuccess
}

func (g *GroupSock) HandleRead(buffer []byte) (int, error) {
	numBytes, err := ReadSocket(g.socketNum, buffer)
	if err != nil {
		fmt.Printf("GroupSock read failed: %s\n", err.Error())
		return numBytes, err
	}

	return numBytes, err
}

func (g *GroupSock) GetSourcePort() uint {
	if g.socketNum != nil {
		localAddr := strings.Split(g.socketNum.LocalAddr().String(), ":")
		sourcePort, err := strconv.Atoi(localAddr[1])
		if err == nil {
			return uint(sourcePort)
		}
	}
	return 0
}

func (g *GroupSock) TTL() uint {
	return g.ttl
}

func (g *GroupSock) AddDestination(addr string, port uint) {
	g.dests = append(g.dests, NewDestRecord(addr, port))
}

func (g *GroupSock) delDestination(addr string, port uint) {
}

type destRecord struct {
	addrStr string
	portNum uint
}

func NewDestRecord(addr string, port uint) *destRecord {
	dest := new(destRecord)
	dest.addrStr = addr
	dest.portNum = port
	return dest
}
