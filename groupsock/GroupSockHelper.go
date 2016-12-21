package rtspclient

import (
	"fmt"
	"net"
)

func InitWinSocket() {
}

func SetupDatagramSocket(address string, port uint) *net.UDPConn {
	addr := fmt.Sprintf("%s:%d", address, port)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		fmt.Println("Failed to resolve UDP address.", err)
		return nil
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Failed to listen UDP address.", err)
		return nil
	}

	return udpConn
}

func SetupStreamSocket(address string, port uint) *net.TCPConn {
	addr := fmt.Sprintf("%s:%d", address, port)
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		fmt.Println("Failed to resolve TCP address.", err)
		return nil
	}

	tcpConn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil
	}
	return tcpConn
}

func createSocket(sockType string) {
}

func closeSocket() {
}

func ReadSocket(conn net.Conn, buffer []byte) (int, error) {
	return conn.Read(buffer)
}

func writeSocket(conn net.Conn, buffer []byte) bool {
	_, err := conn.Write(buffer)
	if err != nil {
		//fmt.Println(writeBytes)
		return false
	}

	return true
}
