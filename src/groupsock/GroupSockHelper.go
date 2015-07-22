package groupsock

import (
	"fmt"
	"net"
	"strings"
)

func InitWinSocket() {
}

func SetupDatagramSocket() {
}

func SetupStreamSocket() {
}

func createSocket(sockType string) {
}

func readSocket() {
}

func writeSocket(address string, port int, buffer []byte, bufferSize int) bool {
    udpConn, err := net.DialUDP("udp", nil, addr)
    if err != nil {
        return false
    }

    writeBytes, err = udpConn.WriteTo(buffer, addr)
    if err != nil {
        fmt.Println(writeBytes)
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
