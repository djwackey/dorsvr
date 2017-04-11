package livemedia

import (
	"fmt"
)

var transportSyncByte byte = 0x47

type PIDStatus struct {
	firstClock, lastClock, firstRealTime, lastRealTime float32
	lastPacketNum                                      uint
}

func NewPIDStatus() *PIDStatus {
	return new(PIDStatus)
}

type M2TSVideoStreamFramer struct {
	FramedFilter
	pcrLimit                    float32
	tsPCRCount                  uint
	tsPacketCount               uint
	numTSPacketsToStream        uint
	limitNumTSPacketsToStream   bool
	limitTSPacketsToStreamByPCR bool
	pidStatusDict               map[byte]*PIDStatus
}

func NewM2TSVideoStreamFramer(inputSource IFramedSource) *M2TSVideoStreamFramer {
	return new(M2TSVideoStreamFramer)
}

func (f *M2TSVideoStreamFramer) doGetNextFrame() error {
	if f.limitNumTSPacketsToStream {
		if f.numTSPacketsToStream == 0 {
			//f.handleClosure(this)
			return nil
		}
		if f.numTSPacketsToStream*TRANSPORT_PACKET_SIZE < f.maxSize {
			f.maxSize = f.numTSPacketsToStream * TRANSPORT_PACKET_SIZE
		}
	}
	return nil
}

func (f *M2TSVideoStreamFramer) doStopGettingFrames() error {
	f.tsPacketCount = 0
	f.tsPCRCount = 0

	return f.clearPIDStatusTable()
}

func (f *M2TSVideoStreamFramer) afterGettingFrame() {
}

func (f *M2TSVideoStreamFramer) setNumTSPacketsToStream(numTSRecordsToStream uint) {
	f.numTSPacketsToStream = numTSRecordsToStream
	if numTSRecordsToStream > 0 {
		f.limitNumTSPacketsToStream = true
	} else {
		f.limitNumTSPacketsToStream = false
	}
}

func (f *M2TSVideoStreamFramer) clearPIDStatusTable() error {
	return nil
}

func (f *M2TSVideoStreamFramer) updateTSPacketDurationEstimate(pkt []byte, timeNow float32) bool {
	if pkt[0] == transportSyncByte {
		fmt.Println("Missing sync byte!")
		return false
	}
	f.tsPacketCount++

	// If this packet doesn't contain a PCR, then we're not interested in it:
	adaptation_field_control := (pkt[3] & 0x30) >> 4
	if adaptation_field_control != 2 && adaptation_field_control != 3 {
		// there's no adaptation_field
		return false
	}

	adaptation_field_length := pkt[4]
	if adaptation_field_length == 0 {
		return false
	}

	//discontinuity_indicator := pkt[5]&0x80
	pcrFlag := pkt[5] & 0x10
	if pcrFlag == 0 {
		// no PCR
		return false
	}

	// There's a PCR.  Get it, and the PID:
	f.tsPCRCount++
	pcrBaseHigh := float32((pkt[6] << 24) | (pkt[7] << 16) | (pkt[8] << 8) | pkt[9])
	clock := pcrBaseHigh / 45000.0
	if (pkt[10] & 0x80) != 0 {
		clock += 1 / 90000.0 // add in low-bit (if set)
	}
	pcrExt := float32(((pkt[10] & 0x01) << 8) | pkt[11])
	clock += pcrExt / 27000000.0
	if f.limitTSPacketsToStreamByPCR {
		if clock > f.pcrLimit {
			// We've hit a preset limit within the stream:
			return false
		}
	}

	pid := ((pkt[1] & 0x1F) << 8) | pkt[2]
	pidStatus := f.pidStatusDict[pid]
	if pidStatus == nil {
		pidStatus = NewPIDStatus()
	}

	pidStatus.lastClock = clock
	pidStatus.lastRealTime = timeNow
	pidStatus.lastPacketNum = f.tsPacketCount
	return true
}
