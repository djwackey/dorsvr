package liveMedia

import (
	"fmt"
	"testing"
)

func Test_enqueueWord(t *testing.T) {
	var preferredPacketSize, maxPacketSize uint = 1000, 1448
	outBuf := NewOutPacketBuffer(preferredPacketSize, maxPacketSize)

	seqNo, rtpPayloadType := uint(2), uint(96)

	var rtpHdr uint = 0x80000000
	rtpHdr |= rtpPayloadType << 16
	rtpHdr |= seqNo

	outBuf.enqueueWord(rtpHdr)

	packet := outBuf.packet()[:200]
	fmt.Println("enqueueWord", packet)
}
