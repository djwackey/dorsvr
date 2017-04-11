package groupsock

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/djwackey/dorsvr/log"
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

func (o *OutputSocket) write(destAddr string, portNum uint, buffer []byte, bufferSize uint) (int, error) {
	addr := fmt.Sprintf("%s:%d", destAddr, portNum)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Error(1, "[OutputSocket::write] Failed to resolve UDP address.%s", err.Error())
		return 0, err
	}
	return o.socketNum.(*net.UDPConn).WriteToUDP(buffer[:bufferSize], udpAddr)
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
	return gs
}

func (g *GroupSock) Output(buffer []byte, bufferSize, ttlToSend uint) bool {
	var err error
	var writeSuccess bool
	for _, dest := range g.dests {
		if _, err = g.write(dest.addrStr, dest.portNum, buffer, bufferSize); err == nil {
			writeSuccess = true
		}
	}
	return writeSuccess
}

func (g *GroupSock) HandleRead(buffer []byte) (int, error) {
	numBytes, err := ReadSocket(g.socketNum, buffer)
	if err != nil {
		log.Error(1, "[GroupSock::HandleRead] read failed: %s\n", err.Error())
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
	g.dests = append(g.dests, newDestRecord(addr, port))
}

func (g *GroupSock) delDestination() {
}

type destRecord struct {
	addrStr string
	portNum uint
}

func newDestRecord(addr string, port uint) *destRecord {
	return &destRecord{
		addrStr: addr,
		portNum: port,
	}
}
