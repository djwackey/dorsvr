package livemedia

import (
	sys "syscall"
)

//////// RTPTransmissionStatsDB ////////
type RTPTransmissionStatsDB struct {
	sink  *RTPSink
	table map[uint32]*RTPTransmissionStats
}

func newRTPTransmissionStatsDB(sink *RTPSink) *RTPTransmissionStatsDB {
	return &RTPTransmissionStatsDB{
		sink:  sink,
		table: make(map[uint32]*RTPTransmissionStats),
	}
}

func (d *RTPTransmissionStatsDB) add(ssrc uint32, s *RTPTransmissionStats) {
	d.table[ssrc] = s
}

func (d *RTPTransmissionStatsDB) lookup(ssrc uint32) *RTPTransmissionStats {
	s, _ := d.table[ssrc]
	return s
}

func (d *RTPTransmissionStatsDB) noteIncomingRR(lastFromAddress string,
	ssrc, lossStats, lastPacketNumReceived, jitter, lastSRTime, diffSRRRTime uint32) {
	stats := d.lookup(ssrc)
	if stats == nil {
		// This is the first time we've heard of this SSRC.
		// Create a new record for it:
		stats = newRTPTransmissionStats(d.sink, ssrc)
		if stats == nil {
			return
		}
		d.add(ssrc, stats)
	}

	stats.noteIncomingRR(lastFromAddress,
		lossStats, lastPacketNumReceived, jitter,
		lastSRTime, diffSRRRTime)
}

//////// RTPTransmissionStats ////////
type RTPTransmissionStats struct {
	sink                          *RTPSink
	ssrc                          uint32
	jitter                        uint32
	lastSRTime                    uint32
	diffSRRRTime                  uint32
	lastOctetCount                uint32
	lastPacketCount               uint32
	packetLossRatio               uint32
	totNumPacketsLost             uint32
	totalOctetCountLo             uint32
	totalOctetCountHi             uint32
	totalPacketCountLo            uint32
	totalPacketCountHi            uint32
	lastPacketNumReceived         uint32
	firstPacketNumReported        uint32
	oldLastPacketNumReceived      uint32
	oldTotNumPacketsLost          uint32
	lastFromAddress               string
	atLeastTwoRRsHaveBeenReceived bool
	firstPacket                   bool
	timeCreated                   sys.Timeval
	timeReceived                  sys.Timeval
}

func newRTPTransmissionStats(sink *RTPSink, ssrc uint32) *RTPTransmissionStats {
	return &RTPTransmissionStats{
		ssrc: ssrc,
		sink: sink,
	}
}

func (s *RTPTransmissionStats) noteIncomingRR(lastFromAddress string,
	lossStats, lastPacketNumReceived, jitter, lastSRTime, diffSRRRTime uint32) {
	if s.firstPacket {
		s.firstPacket = false
		s.firstPacketNumReported = lastPacketNumReceived
	} else {
		s.atLeastTwoRRsHaveBeenReceived = true
		s.oldLastPacketNumReceived = s.lastPacketNumReceived
		s.oldTotNumPacketsLost = s.totNumPacketsLost
	}
	sys.Gettimeofday(&s.timeReceived)

	s.lastFromAddress = lastFromAddress
	s.packetLossRatio = lossStats >> 24
	s.totNumPacketsLost = lossStats & 0xFFFFFF
	s.lastPacketNumReceived = lastPacketNumReceived
	s.jitter = jitter
	s.lastSRTime = lastSRTime
	s.diffSRRRTime = diffSRRRTime

	// Update our counts of the total number of octets and packets sent towards
	// this receiver:
	newOctetCount := uint32(s.sink.octetCount())
	octetCountDiff := newOctetCount - s.lastOctetCount
	s.lastOctetCount = newOctetCount
	prevTotalOctetCountLo := s.totalOctetCountLo
	s.totalOctetCountLo += octetCountDiff
	if s.totalOctetCountLo < prevTotalOctetCountLo { // wrap around
		s.totalOctetCountHi++
	}

	newPacketCount := uint32(s.sink.packetCount())
	packetCountDiff := newPacketCount - s.lastPacketCount
	s.lastPacketCount = newPacketCount
	prevTotalPacketCountLo := s.totalPacketCountLo
	s.totalPacketCountLo += packetCountDiff
	if s.totalPacketCountLo < prevTotalPacketCountLo { // wrap around
		s.totalPacketCountHi++
	}
}
