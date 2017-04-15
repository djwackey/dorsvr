package groupsock

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"net"
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
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Printf("Failed to get InterfaceAddrs.%s\n", err.Error())
		return "", err
	}

	var ip string
	err = errors.New("ip address not found")
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip, err = ipnet.IP.String(), nil
				//break
			}
		}
	}
	return ip, err
}
