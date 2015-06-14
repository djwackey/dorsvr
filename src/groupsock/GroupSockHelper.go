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

func readSocket() {
}

func writeSocket() {
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
