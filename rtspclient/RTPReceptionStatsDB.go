package rtspclient

type RTPReceptionStats struct {
	SSRC                             uint32
	syncTimestamp                    uint32
	lastReceivedSR_NTPmsw            uint32
	lastReceivedSR_NTPlsw            uint32
	numPacketsReceivedSinceLastReset uint32
	lastReceivedSR_time              Timeval
	syncTime                         Timeval
	hasBeenSynchronized              bool
}

func NewRTPReceptionStats(SSRC uint32) *RTPReceptionStats {
	stats := new(RTPReceptionStats)
	stats.SSRC = SSRC
	return stats
}

func (stats *RTPReceptionStats) noteIncomingPacket(seqNum, rtpTimestamp, timestampFrequency, packetSize uint32,
	useForJitterCalculation bool) {
}

func (stats *RTPReceptionStats) noteIncomingSR(ntpTimestampMSW, ntpTimestampLSW, rtpTimestamp uint32) {
	stats.lastReceivedSR_NTPmsw = ntpTimestampMSW
	stats.lastReceivedSR_NTPlsw = ntpTimestampLSW

	GetTimeOfDay(&stats.lastReceivedSR_time)

	// Use this SR to update time synchronization information:
	stats.syncTimestamp = rtpTimestamp
	stats.syncTime.Tv_sec = int64(ntpTimestampMSW - 0x83AA7E80)       // 1/1/1900 -> 1/1/1970
	microseconds := float32((ntpTimestampLSW * 15625.0) / 0x04000000) // 10^6/2^32
	stats.syncTime.Tv_usec = int64(microseconds + 0.5)
	stats.hasBeenSynchronized = true
}

func (stats *RTPReceptionStats) NumPacketsReceivedSinceLastReset() uint32 {
	return stats.numPacketsReceivedSinceLastReset
}

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

func (statsDB *RTPReceptionStatsDB) noteIncomingPacket(SSRC, seqNum, rtpTimestamp, timestampFrequency, packetSize uint32,
	useForJitterCalculation bool) (presentationTime Timeval, hasBeenSyncedUsingRTCP bool) {
	statsDB.totNumPacketsReceived++

	stats := statsDB.lookup(SSRC)
	if stats == nil {
		stats = NewRTPReceptionStats(SSRC)
		if stats == nil {
			return
		}

		statsDB.add(SSRC, stats)
	}

	if stats.NumPacketsReceivedSinceLastReset() == 0 {
		statsDB.numActiveSourcesSinceLastReset++
	}

	stats.noteIncomingPacket(seqNum, rtpTimestamp, timestampFrequency, packetSize, useForJitterCalculation)
	return
}

func (statsDB *RTPReceptionStatsDB) noteIncomingSR(SSRC, ntpTimestampMSW, ntpTimestampLSW, rtpTimestamp uint32) {
	stats := statsDB.lookup(SSRC)
	if stats == nil {
		stats = NewRTPReceptionStats(SSRC)
		if stats == nil {
			return
		}
		statsDB.table[SSRC] = stats
	}
	stats.noteIncomingSR(ntpTimestampMSW, ntpTimestampLSW, rtpTimestamp)
}
