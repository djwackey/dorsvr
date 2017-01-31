package livemedia

import sys "syscall"

////////// RTPReceptionStats //////////
const MILLION = 1000000

type RTPReceptionStats struct {
	SSRC                             uint32
	syncTimestamp                    uint32
	totBytesReceived_hi              uint32
	totBytesReceived_lo              uint32
	minInterPacketGapUS              uint32
	maxInterPacketGapUS              uint32
	lastReceivedSR_NTPmsw            uint32
	lastReceivedSR_NTPlsw            uint32
	baseExtSeqNumReceived            uint32
	totNumPacketsReceived            uint32 // for all SSRCs
	highestExtSeqNumReceived         uint32
	previousPacketRTPTimestamp       uint32
	lastResetExtSeqNumReceived       uint32
	numPacketsReceivedSinceLastReset uint32
	syncTime                         sys.Timeval
	lastReceivedSR_time              sys.Timeval
	lastPacketReceptionTime          sys.Timeval
	totalInterPacketGaps             sys.Timeval
	hasBeenSynchronized              bool
	haveSeenInitialSequenceNumber    bool
	lastTransit                      int
	jitter                           float32
}

func NewRTPReceptionStats(SSRC, seqNum uint32) *RTPReceptionStats {
	stats := new(RTPReceptionStats)
	stats.init(SSRC)
	stats.initSeqNum(seqNum)
	return stats
}

func (stats *RTPReceptionStats) init(SSRC uint32) {
	stats.SSRC = SSRC
	stats.totNumPacketsReceived = 0
	stats.totBytesReceived_hi = 0
	stats.totBytesReceived_lo = 0
	stats.haveSeenInitialSequenceNumber = false
	stats.lastTransit = -1
	stats.previousPacketRTPTimestamp = 0
	stats.jitter = 0.0
	stats.lastReceivedSR_NTPmsw = 0
	stats.lastReceivedSR_NTPlsw = 0
	stats.lastReceivedSR_time.Sec = 0
	stats.lastReceivedSR_time.Usec = 0
	stats.lastPacketReceptionTime.Sec = 0
	stats.lastPacketReceptionTime.Usec = 0
	stats.minInterPacketGapUS = 0x7FFFFFFF
	stats.maxInterPacketGapUS = 0
	stats.totalInterPacketGaps.Sec = 0
	stats.totalInterPacketGaps.Usec = 0
	stats.hasBeenSynchronized = false
	stats.syncTime.Sec = 0
	stats.syncTime.Usec = 0
	stats.reset()
}

func (stats *RTPReceptionStats) reset() {
	stats.numPacketsReceivedSinceLastReset = 0
	stats.lastResetExtSeqNumReceived = stats.highestExtSeqNumReceived
}

func (stats *RTPReceptionStats) initSeqNum(initialSeqNum uint32) {
	stats.baseExtSeqNumReceived = 0x10000 | initialSeqNum
	stats.highestExtSeqNumReceived = 0x10000 | initialSeqNum
	stats.haveSeenInitialSequenceNumber = true
}

func (stats *RTPReceptionStats) noteIncomingPacket(seqNum, rtpTimestamp, timestampFrequency, packetSize uint32,
	useForJitterCalculation bool) (resultPresentationTime sys.Timeval, resultHasBeenSyncedUsingRTCP bool) {
	if !stats.haveSeenInitialSequenceNumber {
		stats.initSeqNum(seqNum)
	}

	stats.numPacketsReceivedSinceLastReset++
	stats.totNumPacketsReceived++

	prevTotBytesReceived_lo := stats.totBytesReceived_lo
	stats.totBytesReceived_lo += packetSize
	if stats.totBytesReceived_lo < prevTotBytesReceived_lo { // wrap-around
		stats.totBytesReceived_hi++
	}

	// Check whether the new sequence number is the highest yet seen:
	oldSeqNum := (stats.highestExtSeqNumReceived & 0xFFFF)
	seqNumCycle := (stats.highestExtSeqNumReceived & 0xFFFF0000)
	seqNumDifference := seqNum - oldSeqNum
	var newSeqNum uint32

	if seqNumLT(int(oldSeqNum), int(seqNum)) {
		// This packet was not an old packet received out of order, so check it:

		if seqNumDifference >= 0x8000 {
			// The sequence number wrapped around, so start a new cycle:
			seqNumCycle += 0x10000
		}

		newSeqNum = seqNumCycle | seqNum
		if newSeqNum > stats.highestExtSeqNumReceived {
			stats.highestExtSeqNumReceived = newSeqNum
		}
	} else if stats.totNumPacketsReceived > 1 {
		// This packet was an old packet received out of order

		if seqNumDifference >= 0x8000 {
			// The sequence number wrapped around, so switch to an old cycle:
			seqNumCycle -= 0x10000
		}

		newSeqNum = seqNumCycle | seqNum
		if newSeqNum < stats.baseExtSeqNumReceived {
			stats.baseExtSeqNumReceived = newSeqNum
		}
	}

	// Record the inter-packet delay
	var timeNow sys.Timeval
	sys.Gettimeofday(&timeNow)
	if stats.lastPacketReceptionTime.Sec != 0 ||
		stats.lastPacketReceptionTime.Usec != 0 {
		gap := (timeNow.Sec-stats.lastPacketReceptionTime.Sec)*MILLION +
			timeNow.Usec - stats.lastPacketReceptionTime.Usec
		if gap > int32(stats.maxInterPacketGapUS) {
			stats.maxInterPacketGapUS = uint32(gap)
		}
		if gap < int32(stats.minInterPacketGapUS) {
			stats.minInterPacketGapUS = uint32(gap)
		}
		stats.totalInterPacketGaps.Usec += gap
		if stats.totalInterPacketGaps.Usec >= MILLION {
			stats.totalInterPacketGaps.Sec++
			stats.totalInterPacketGaps.Usec -= MILLION
		}
	}
	stats.lastPacketReceptionTime = timeNow

	// Compute the current 'jitter' using the received packet's RTP timestamp,
	// and the RTP timestamp that would correspond to the current time.
	// (Use the code from appendix A.8 in the RTP spec.)
	// Note, however, that we don't use this packet if its timestamp is
	// the same as that of the previous packet (this indicates a multi-packet
	// fragment), or if we've been explicitly told not to use this packet.
	if useForJitterCalculation && rtpTimestamp != stats.previousPacketRTPTimestamp {
		arrival := int32(timestampFrequency) * timeNow.Sec
		arrival += ((2.0*int32(timestampFrequency)*timeNow.Usec + 1000000.0) / 2000000)
		// note: rounding
		transit := arrival - int32(rtpTimestamp)
		if stats.lastTransit == -1 {
			stats.lastTransit = int(transit) // hack for first time
		}
		d := transit - int32(stats.lastTransit)
		stats.lastTransit = int(transit)
		if d < 0 {
			d = -d
		}
		stats.jitter += (1.0 / 16.0) * (float32(d) - stats.jitter)
	}

	// Return the 'presentation time' that corresponds to "rtpTimestamp":
	if stats.syncTime.Sec == 0 && stats.syncTime.Usec == 0 {
		// This is the first timestamp that we've seen, so use the current
		// 'wall clock' time as the synchronization time.  (This will be
		// corrected later when we receive RTCP SRs.)
		stats.syncTimestamp = rtpTimestamp
		stats.syncTime = timeNow
	}

	timestampDiff := rtpTimestamp - stats.syncTimestamp
	// Note: This works even if the timestamp wraps around
	// (as long as "int" is 32 bits)

	// Divide this by the timestamp frequency to get real time:
	timeDiff := float32(timestampDiff) / float32(timestampFrequency)

	// Add this to the 'sync time' to get our result:
	var million float32 = 1000000
	var seconds, uSeconds int32
	if timeDiff >= 0.0 {
		seconds = stats.syncTime.Sec + int32(timeDiff)
		uSeconds = stats.syncTime.Usec + int32((timeDiff-float32(int32(timeDiff)))*million)
		if uSeconds >= int32(million) {
			uSeconds -= int32(million)
			seconds++
		}
	} else {
		timeDiff = -timeDiff
		seconds = stats.syncTime.Sec - int32(timeDiff)
		uSeconds = stats.syncTime.Usec - int32((timeDiff-float32(int32(timeDiff)))*million)
		if uSeconds < 0 {
			uSeconds += int32(million)
			seconds--
		}
	}

	resultPresentationTime.Sec = int32(seconds)
	resultPresentationTime.Usec = int32(uSeconds)
	resultHasBeenSyncedUsingRTCP = stats.hasBeenSynchronized

	// Save these as the new synchronization timestamp & time:
	stats.syncTimestamp = rtpTimestamp
	stats.syncTime = resultPresentationTime

	stats.previousPacketRTPTimestamp = rtpTimestamp
	return
}

func (stats *RTPReceptionStats) noteIncomingSR(ntpTimestampMSW, ntpTimestampLSW, rtpTimestamp uint32) {
	stats.lastReceivedSR_NTPmsw = ntpTimestampMSW
	stats.lastReceivedSR_NTPlsw = ntpTimestampLSW

	sys.Gettimeofday(&stats.lastReceivedSR_time)

	// Use this SR to update time synchronization information:
	stats.syncTimestamp = rtpTimestamp
	stats.syncTime.Sec = int32(ntpTimestampMSW - 0x83AA7E80)          // 1/1/1900 -> 1/1/1970
	microseconds := float32((ntpTimestampLSW * 15625.0) / 0x04000000) // 10^6/2^32
	stats.syncTime.Usec = int32(microseconds + 0.5)
	stats.hasBeenSynchronized = true
}

func (stats *RTPReceptionStats) NumPacketsReceivedSinceLastReset() uint32 {
	return stats.numPacketsReceivedSinceLastReset
}

////////// RTPReceptionStatsDB //////////

type RTPReceptionStatsDB struct {
	table                          map[uint32]*RTPReceptionStats
	totNumPacketsReceived          uint32
	numActiveSourcesSinceLastReset uint32
}

func NewRTPReceptionStatsDB() *RTPReceptionStatsDB {
	statsDB := new(RTPReceptionStatsDB)
	statsDB.table = make(map[uint32]*RTPReceptionStats)
	return statsDB
}

func (statsDB *RTPReceptionStatsDB) add(SSRC uint32, stats *RTPReceptionStats) {
	statsDB.table[SSRC] = stats
}

func (statsDB *RTPReceptionStatsDB) lookup(SSRC uint32) *RTPReceptionStats {
	stats, _ := statsDB.table[SSRC]
	return stats
}

func (statsDB *RTPReceptionStatsDB) noteIncomingPacket(SSRC, seqNum,
	rtpTimestamp, timestampFrequency, packetSize uint32,
	useForJitterCalculation bool) (presentationTime sys.Timeval, hasBeenSyncedUsingRTCP bool) {
	statsDB.totNumPacketsReceived++

	stats := statsDB.lookup(SSRC)
	if stats == nil {
		stats = NewRTPReceptionStats(SSRC, seqNum)
		if stats == nil {
			return
		}

		statsDB.add(SSRC, stats)
	}

	if stats.NumPacketsReceivedSinceLastReset() == 0 {
		statsDB.numActiveSourcesSinceLastReset++
	}

	presentationTime, hasBeenSyncedUsingRTCP = stats.noteIncomingPacket(seqNum, rtpTimestamp,
		timestampFrequency, packetSize, useForJitterCalculation)
	return
}

func (statsDB *RTPReceptionStatsDB) noteIncomingSR(SSRC, ntpTimestampMSW, ntpTimestampLSW, rtpTimestamp uint32) {
	stats := statsDB.lookup(SSRC)
	if stats == nil {
		stats = NewRTPReceptionStats(SSRC, 0)
		if stats == nil {
			return
		}
		statsDB.table[SSRC] = stats
	}
	stats.noteIncomingSR(ntpTimestampMSW, ntpTimestampLSW, rtpTimestamp)
}
