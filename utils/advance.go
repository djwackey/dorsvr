package utils

func Advance(data []byte, size, n uint) ([]byte, uint) {
	data = data[n:]
	size -= n
	return data, size
}

func SeqNumLT(s1, s2 int) bool {
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
