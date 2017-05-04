package groupsock

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// GroupSock is used to both send and receive packets.
// As the name suggests, it was originally designed to send/receive
// multicast, but it can send/receive unicast as well.
type GroupSock struct {
	portNum uint
	udpConn *net.UDPConn
	dests   []*destRecord
}

// NewGroupSock returns a source-independent multicast group
func NewGroupSock(addrStr string, portNum uint) *GroupSock {
	udpConn := SetupDatagramSocket(addrStr, portNum)
	if udpConn == nil {
		return nil
	}
	return &GroupSock{
		portNum: portNum,
		udpConn: udpConn,
	}
}

// Output does the datagram send, to each destination.
func (g *GroupSock) Output(buffer []byte, bufferSize uint) bool {
	var err error
	var writeSuccess bool
	for _, dest := range g.dests {
		if _, err = g.write(dest.addrStr, dest.portNum, buffer, bufferSize); err == nil {
			writeSuccess = true
		}
	}
	return writeSuccess
}

func (g *GroupSock) write(destAddr string, portNum uint, buffer []byte, bufferSize uint) (int, error) {
	addr := fmt.Sprintf("%s:%d", destAddr, portNum)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return 0, err
	}

	return g.udpConn.WriteToUDP(buffer[:bufferSize], udpAddr)
}

// Close is responsible for disconnection from client.
func (g *GroupSock) Close() {
	g.udpConn.Close()
}

// HandleRead reads data from client connection.
func (g *GroupSock) HandleRead(buffer []byte) (int, error) {
	numBytes, err := ReadSocket(g.udpConn, buffer)
	if err != nil {
		return numBytes, err
	}

	return numBytes, err
}

// GetSourcePort returns the source port of system allocation.
func (g *GroupSock) GetSourcePort() uint {
	if g.udpConn != nil {
		localAddr := strings.Split(g.udpConn.LocalAddr().String(), ":")
		sourcePort, err := strconv.Atoi(localAddr[1])
		if err == nil {
			return uint(sourcePort)
		}
	}
	return 0
}

// AddDestination can add multiple destinations (addresses & ports)
// This can be used to implement multi-unicast.
func (g *GroupSock) AddDestination(addr string, port uint) {
	g.dests = append(g.dests, newDestRecord(addr, port))
}

// DelDestination can remove the destinations.
func (g *GroupSock) DelDestination() {
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
