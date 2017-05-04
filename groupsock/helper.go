package groupsock

import (
	"fmt"
	"net"
)

// SetupDatagramSocket returns a udp connection of Listening to the specified port.
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

	//go func() {
	//	data := make([]byte, 32)
	//	for {
	//		n, remoteAddr, err := socketNum.ReadFromUDP(data)
	//		if err != nil {
	//			fmt.Printf("error during read: %v\n", err)
	//			break
	//		}

	//		fmt.Printf("<%s> %s\n", remoteAddr, data[:n])
	//	}
	//}()

	return udpConn
}

func setupStreamSocket(address string, port uint) *net.TCPConn {
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

// ReadSocket reads data from the connection.
func ReadSocket(conn net.Conn, buffer []byte) (int, error) {
	return conn.Read(buffer)
}

func writeSocket(conn net.Conn, buffer []byte) (int, error) {
	return conn.Write(buffer)
}
