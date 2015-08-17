package groupsock

import (
	"fmt"
	"net"
	"strings"
)

func InitWinSocket() {
}

func SetupDatagramSocket(address string, port uint) *net.Conn {
	addr := fmt.Sprintf("%s:%d", address, port)
	udpAddr, _ := net.ResolveUDPAddr("udp", addr)

	udpConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil
	}
    return udpConn
}

func SetupStreamSocket() {
}

func createSocket(sockType string) {
}

func closeSocket() {
}

func readSocket(udpConn *net.Conn, buffer []byte) int {
    readBytes, _, _ := udpConn.ReadFrom(buffer)
    return readBytes
}

func writeSocket(udpConn *net.Conn, udpAddr Addr, buffer []byte) bool {
	_, err = udpConn.WriteTo(buffer, udpAddr)
	if err != nil {
		//fmt.Println(writeBytes)
		return false
	}

	return true
}

func OurIPAddress() (string, error) {
	conn, err := net.Dial("udp", "www.baidu.com:80")
	if err != nil {
		fmt.Println("[ourIPAddress]", err.Error())
		return "", err
	}
	defer conn.Close()

	return strings.Split(conn.LocalAddr().String(), ":")[0], nil
}
