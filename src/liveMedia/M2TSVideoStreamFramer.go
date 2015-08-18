package liveMedia

import (
    "fmt"
)

var TRANSPORT_PACKET_SIZE = 188
var TRANSPORT_SYNC_BYTE = 0x47

type PIDStatus struct {
    firstClock, lastClock, firstRealTime, lastRealTime float32
    lastPacketNum uint
}

func NewPIDStatus() *PIDStatus {
    return new(PIDStatus)
}

type M2TSVideoStreamFramer struct {
    FramedFilter
    pcrLimit float32
    tsPCRCount uint
    tsPacketCount uint
    numTSPacketsToStream uint
    limitNumTSPacketsToStream bool
    limitTSPacketsToStreamByPCR bool
    pidStatusList []PIDStatus
}

func (this *M2TSVideoStreamFramer) doGetNextFrame() {
    if this.limitNumTSPacketsToStream {
        if this.numTSPacketsToStream == 0 {
            this.handleClosure(this)
            return
        }
        if this.numTSPacketsToStream * TRANSPORT_PACKET_SIZE < this.maxSize {
            this.maxSize = this.numTSPacketsToStream * TRANSPORT_PACKET_SIZE
        }
    }
}

func (this *M2TSVideoStreamFramer) doStopGettingFrames() {
    //FramedFilter::doStopGettingFrames()
    this.tsPacketCount = 0
    this.tsPCRCount = 0

    this.clearPIDStatusTable()
}

func (this *M2TSVideoStreamFramer) afterGettingFrame() {
}

func (this *M2TSVideoStreamFramer) setNumTSPacketsToStream(numTSRecordsToStream uint) {
    this.numTSPacketsToStream = numTSRecordsToStream
    if numTSRecordsToStream > 0 {
        this.limitNumTSPacketsToStream = true
    } else {
        this.limitNumTSPacketsToStream = false
    }
}

func (this *M2TSVideoStreamFramer) clearPIDStatusTable() {
}

func (this *M2TSVideoStreamFramer) updateTSPacketDurationEstimate(pkt []byte, timeNow float32) bool {
    if pkt[0] == TRANSPORT_SYNC_BYTE {
        fmt.Println("Missing sync byte!")
        return false
    }
    this.tsPacketCount++

    // If this packet doesn't contain a PCR, then we're not interested in it:
    adaptation_field_control := (pkt[3]&0x30)>>4
    if adaptation_field_control != 2 && adaptation_field_control != 3 {
        // there's no adaptation_field
        return false
    }

    adaptation_field_length := pkt[4]
    if adaptation_field_length == 0 {
        return false
    }

    discontinuity_indicator := pkt[5]&0x80
    pcrFlag := pkt[5]&0x10
    if pcrFlag == 0 {
        // no PCR
        return false
    }

    // There's a PCR.  Get it, and the PID:
    this.tsPCRCount++
    pcrBaseHigh := (pkt[6]<<24)|(pkt[7]<<16)|(pkt[8]<<8)|pkt[9]
    clock := pcrBaseHigh / 45000.0
    if (pkt[10]&0x80) != 0 {
        clock += 1/90000.0  // add in low-bit (if set)
    }
    pcrExt = ((pkt[10]&0x01)<<8) | pkt[11]
    clock += pcrExt/27000000.0
    if this.limitTSPacketsToStreamByPCR {
        if clock > this.pcrLimit {
            // We've hit a preset limit within the stream:
            return false
        }
    }

    pid := ((pkt[1]&0x1F)<<8) | pkt[2]
    pidStatus := pidStatusDict[pid]
    if pidStatus == nil {
        pidStatus = NewPIDStatus()
    }

    pidStatus.lastClock = clock
    pidStatus.lastRealTime = timeNow
    pidStatus.lastPacketNum = this.tsPacketCount
}
