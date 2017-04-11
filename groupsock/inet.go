package groupsock

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"
)

func OurRandom32() uint32 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	random16_1 := r.Int31() & 0x00FFFF00
	random16_2 := r.Int31() & 0x00FFFF00
	return uint32((random16_1 << 8) | (random16_2 >> 8))
}

func OurRandom16() uint32 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return uint32(r.Int31() >> 16)
}

func Ntohl(packet []byte) (uint32, error) {
	var value uint32
	buffer := bytes.NewReader(packet)

	err := binary.Read(buffer, binary.BigEndian, &value)
	if err != nil {
		fmt.Println("failed to read binary.", err.Error())
		return value, err
	}

	return value, nil
}

func OurIPAddress() (string, error) {
	conn, err := net.Dial("udp", "www.baidu.com:80")
	if err != nil {
		fmt.Println("Failed to get our IP address", err.Error())
		return "", err
	}
	defer conn.Close()

	return strings.Split(conn.LocalAddr().String(), ":")[0], nil
}
