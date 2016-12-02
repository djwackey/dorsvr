package liveMedia

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

func ADVANCE(data []byte, size, n uint) ([]byte, uint) {
	data = data[n:]
	size -= n
	return data, size
}

func ntohl(packet []byte) (uint32, error) {
	var value uint32
	buffer := bytes.NewReader(packet)

	err := binary.Read(buffer, binary.BigEndian, &value)
	if err != nil {
		fmt.Println("failed to read binary.", err.Error())
		return value, err
	}

	return value, nil
}

func seqNumLT(s1, s2 int) bool {
	// a 'less-than' on 16-bit sequence numbers
	diff := s2 - s1
	if diff > 0 {
		return (diff < 0x8000)
	} else if diff < 0 {
		return (diff < -0x8000)
	} else { // diff == 0
		return false
	}
}
