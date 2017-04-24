package livemedia

import sys "syscall"

////////// RTPReceptionStats //////////
const Million = 1000000

type RTPReceptionStats struct {
	ssrc                             uint32
	syncTimestamp                    uint32
	totBytesReceivedHi               uint32
	totBytesReceivedLo               uint32
	lastReceivedSRNTPmsw             uint32
	lastReceivedSRNTPlsw             uint32
	baseExtSeqNumReceived            uint32
	totNumPacketsReceived            uint32 // for all SSRCs
	highestExtSeqNumReceived         uint32
	previousPacketRTPTimestamp       uint32
	lastResetExtSeqNumReceived       uint32
	numPacketsReceivedSinceLastReset uint32
	minInterPacketGapUS              int64
	maxInterPacketGapUS              int64
	lastTransit                      int64
	syncTime                         sys.Timeval
	lastReceivedSRTime               sys.Timeval
	lastPacketReceptionTime          sys.Timeval
	totalInterPacketGaps             sys.Timeval
	hasBeenSynchronized              bool
	haveSeenInitialSequenceNumber    bool
	jitter                           float32
}

func newRTPReceptionStats(ssrc, seqNum uint32) *RTPReceptionStats {
	s := new(RTPReceptionStats)
	s.init(ssrc)
	s.initSeqNum(seqNum)
	return s
}

func (s *RTPReceptionStats) init(ssrc uint32) {
	s.ssrc = ssrc
	s.totNumPacketsReceived = 0
	s.totBytesReceivedHi = 0
	s.totBytesReceivedLo = 0
	s.haveSeenInitialSequenceNumber = false
	s.lastTransit = -1
	s.previousPacketRTPTimestamp = 0
	s.jitter = 0.0
	s.lastReceivedSRNTPmsw = 0
	s.lastReceivedSRNTPlsw = 0
	s.lastReceivedSRTime.Sec = 0
	s.lastReceivedSRTime.Usec = 0
	s.lastPacketReceptionTime.Sec = 0
	s.lastPacketReceptionTime.Usec = 0
	s.minInterPacketGapUS = 0x7FFFFFFF
	s.maxInterPacketGapUS = 0
	s.totalInterPacketGaps.Sec = 0
	s.totalInterPacketGaps.Usec = 0
	s.hasBeenSynchronized = false
	s.syncTime.Sec = 0
	s.syncTime.Usec = 0
	s.reset()
}

func (s *RTPReceptionStats) reset() {
	s.numPacketsReceivedSinceLastReset = 0
	s.lastResetExtSeqNumReceived = s.highestExtSeqNumReceived
}

func (s *RTPReceptionStats) initSeqNum(initialSeqNum uint32) {
	s.baseExtSeqNumReceived = 0x10000 | initialSeqNum
	s.highestExtSeqNumReceived = 0x10000 | initialSeqNum
	s.haveSeenInitialSequenceNumber = true
}

func (s *RTPReceptionStats) noteIncomingPacket(seqNum, rtpTimestamp, timestampFrequency, packetSize uint32,
	useForJitterCalculation bool) (resultPresentationTime sys.Timeval, resultHasBeenSyncedUsingRTCP bool) {
	if !s.haveSeenInitialSequenceNumber {
		s.initSeqNum(seqNum)
	}

	s.numPacketsReceivedSinceLastReset++
	s.totNumPacketsReceived++

	prevTotBytesReceivedLo := s.totBytesReceivedLo
	s.totBytesReceivedLo += packetSize
	if s.totBytesReceivedLo < prevTotBytesReceivedLo { // wrap-around
		s.totBytesReceivedHi++
	}

	// Check whether the new sequence number is the highest yet seen:
	oldSeqNum := (s.highestExtSeqNumReceived & 0xFFFF)
	seqNumCycle := (s.highestExtSeqNumReceived & 0xFFFF0000)
	seqNumDifference := seqNum - oldSeqNum
	var newSeqNum uint32

	if seqNumLT(int(oldSeqNum), int(seqNum)) {
		// This packet was not an old packet received out of order, so check it:

		if seqNumDifference >= 0x8000 {
			// The sequence number wrapped around, so start a new cycle:
			seqNumCycle += 0x10000
		}

		newSeqNum = seqNumCycle | seqNum
		if newSeqNum > s.highestExtSeqNumReceived {
			s.highestExtSeqNumReceived = newSeqNum
		}
	} else if s.totNumPacketsReceived > 1 {
		// This packet was an old packet received out of order

		if seqNumDifference >= 0x8000 {
			// The sequence number wrapped around, so switch to an old cycle:
			seqNumCycle -= 0x10000
		}

		newSeqNum = seqNumCycle | seqNum
		if newSeqNum < s.baseExtSeqNumReceived {
			s.baseExtSeqNumReceived = newSeqNum
		}
	}

	// Record the inter-packet delay
	var timeNow sys.Timeval
	sys.Gettimeofday(&timeNow)
	if s.lastPacketReceptionTime.Sec != 0 ||
		s.lastPacketReceptionTime.Usec != 0 {
		gap := (timeNow.Sec-s.lastPacketReceptionTime.Sec)*Million +
			timeNow.Usec - s.lastPacketReceptionTime.Usec
		if gap > s.maxInterPacketGapUS {
			s.maxInterPacketGapUS = gap
		}
		if gap < s.minInterPacketGapUS {
			s.minInterPacketGapUS = gap
		}
		s.totalInterPacketGaps.Usec += gap
		if s.totalInterPacketGaps.Usec >= Million {
			s.totalInterPacketGaps.Sec++
			s.totalInterPacketGaps.Usec -= Million
		}
	}
	s.lastPacketReceptionTime = timeNow

	// Compute the current 'jitter' using the received packet's RTP timestamp,
	// and the RTP timestamp that would correspond to the current time.
	// (Use the code from appendix A.8 in the RTP spec.)
	// Note, however, that we don't use this packet if its timestamp is
	// the same as that of the previous packet (this indicates a multi-packet
	// fragment), or if we've been explicitly told not to use this packet.
	if useForJitterCalculation && rtpTimestamp != s.previousPacketRTPTimestamp {
		arrival := int64(timestampFrequency) * timeNow.Sec
		arrival += ((2.0*int64(timestampFrequency)*timeNow.Usec + 1000000.0) / 2000000)
		// note: rounding
		transit := arrival - int64(rtpTimestamp)
		if s.lastTransit == -1 {
			s.lastTransit = transit // hack for first time
		}
		d := transit - s.lastTransit
		s.lastTransit = transit
		if d < 0 {
			d = -d
		}
		s.jitter += (1.0 / 16.0) * (float32(d) - s.jitter)
	}

	// Return the 'presentation time' that corresponds to "rtpTimestamp":
	if s.syncTime.Sec == 0 && s.syncTime.Usec == 0 {
		// This is the first timestamp that we've seen, so use the current
		// 'wall clock' time as the synchronization time.  (This will be
		// corrected later when we receive RTCP SRs.)
		s.syncTimestamp = rtpTimestamp
		s.syncTime = timeNow
	}

	timestampDiff := rtpTimestamp - s.syncTimestamp
	// Note: This works even if the timestamp wraps around
	// (as long as "int" is 32 bits)

	// Divide this by the timestamp frequency to get real time:
	timeDiff := float32(timestampDiff) / float32(timestampFrequency)

	// Add this to the 'sync time' to get our result:
	var million float32 = 1000000
	var seconds, uSeconds int64
	if timeDiff >= 0.0 {
		seconds = s.syncTime.Sec + int64(timeDiff)
		uSeconds = s.syncTime.Usec + int64((timeDiff-float32(int64(timeDiff)))*million)
		if uSeconds >= int64(million) {
			uSeconds -= int64(million)
			seconds++
		}
	} else {
		timeDiff = -timeDiff
		seconds = s.syncTime.Sec - int64(timeDiff)
		uSeconds = s.syncTime.Usec - int64((timeDiff-float32(int64(timeDiff)))*million)
		if uSeconds < 0 {
			uSeconds += int64(million)
			seconds--
		}
	}

	resultPresentationTime.Sec = int64(seconds)
	resultPresentationTime.Usec = int64(uSeconds)
	resultHasBeenSyncedUsingRTCP = s.hasBeenSynchronized

	// Save these as the new synchronization timestamp & time:
	s.syncTimestamp = rtpTimestamp
	s.syncTime = resultPresentationTime

	s.previousPacketRTPTimestamp = rtpTimestamp
	return
}

func (s *RTPReceptionStats) noteIncomingSR(ntpTimestampMSW, ntpTimestampLSW, rtpTimestamp uint32) {
	s.lastReceivedSRNTPmsw = ntpTimestampMSW
	s.lastReceivedSRNTPlsw = ntpTimestampLSW

	sys.Gettimeofday(&s.lastReceivedSRTime)

	// Use this SR to update time synchronization information:
	s.syncTimestamp = rtpTimestamp
	s.syncTime.Sec = int64(ntpTimestampMSW - 0x83AA7E80)              // 1/1/1900 -> 1/1/1970
	microseconds := float32((ntpTimestampLSW * 15625.0) / 0x04000000) // 10^6/2^32
	s.syncTime.Usec = int64(microseconds + 0.5)
	s.hasBeenSynchronized = true
}

////////// RTPReceptionStatsDB //////////
type RTPReceptionStatsDB struct {
	table                          map[uint32]*RTPReceptionStats
	totNumPacketsReceived          uint32
	numActiveSourcesSinceLastReset uint32
}

func newRTPReceptionStatsDB() *RTPReceptionStatsDB {
	return &RTPReceptionStatsDB{
		table: make(map[uint32]*RTPReceptionStats),
	}
}

func (d *RTPReceptionStatsDB) add(ssrc uint32, s *RTPReceptionStats) {
	d.table[ssrc] = s
}

func (d *RTPReceptionStatsDB) lookup(ssrc uint32) *RTPReceptionStats {
	s, _ := d.table[ssrc]
	return s
}

func (d *RTPReceptionStatsDB) noteIncomingPacket(ssrc, seqNum,
	rtpTimestamp, timestampFrequency, packetSize uint32,
	useForJitterCalculation bool) (presentationTime sys.Timeval, hasBeenSyncedUsingRTCP bool) {
	d.totNumPacketsReceived++

	s := d.lookup(ssrc)
	if s == nil {
		s = newRTPReceptionStats(ssrc, seqNum)
		if s == nil {
			return
		}

		d.add(ssrc, s)
	}

	if s.numPacketsReceivedSinceLastReset == 0 {
		d.numActiveSourcesSinceLastReset++
	}

	presentationTime, hasBeenSyncedUsingRTCP = s.noteIncomingPacket(seqNum, rtpTimestamp,
		timestampFrequency, packetSize, useForJitterCalculation)
	return
}

func (d *RTPReceptionStatsDB) noteIncomingSR(ssrc, ntpTimestampMSW, ntpTimestampLSW, rtpTimestamp uint32) {
	s := d.lookup(ssrc)
	if s == nil {
		s = newRTPReceptionStats(ssrc, 0)
		if s == nil {
			return
		}
		d.table[ssrc] = s
	}
	s.noteIncomingSR(ntpTimestampMSW, ntpTimestampLSW, rtpTimestamp)
}
