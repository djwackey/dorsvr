package livemedia

import gs "github.com/djwackey/dorsvr/groupsock"

const (
	eventUnknown = 0
	eventReport  = 1
	eventBye     = 2
)

func drand48() int32 {
	return gs.OurRandom()
}

func rtcpInterval(members, senders, weSent, rtcpBW, avgRtcpSize float64) float64 {
	RTCP_MIN_TIME := 5.
	RTCP_SENDER_BW_FRACTION := 0.25
	RTCP_RCVR_BW_FRACTION := (1 - RTCP_SENDER_BW_FRACTION)
	COMPENSATION := 2.71828 - 1.5

	rtcpMinTime := RTCP_MIN_TIME

	n := members
	if senders > 0 && senders < members*RTCP_SENDER_BW_FRACTION {
		if weSent != 0 {
			rtcpBW *= RTCP_SENDER_BW_FRACTION
			n = senders
		} else {
			rtcpBW *= RTCP_RCVR_BW_FRACTION
			n -= senders
		}
	}

	t := avgRtcpSize * n / rtcpBW
	if t < rtcpMinTime {
		t = rtcpMinTime
	}

	t = t * (float64(drand48()) + 0.5)
	t = t / COMPENSATION
	return t
}

func OnExpire(instance *RTCPInstance, members, senders, weSent, rtcpBW, avgRTCPSize, tc, tp float64) {
	if instance == nil {
		return
	}
	if instance.typeOfEvent == eventBye {
	} else if instance.typeOfEvent == eventReport {
		t := rtcpInterval(members, senders, weSent, rtcpBW, avgRTCPSize)
		tn := tp + t

		if tn <= tc {
			instance.sendReport()
			avgRTCPSize = (1./16.)*float64(instance.sentPacketSize()) + (15./16.)*avgRTCPSize

			t := rtcpInterval(members, senders, weSent, rtcpBW, avgRTCPSize)
			instance.schedule(int64(t + tc))
		} else {
			instance.schedule(int64(tn))
		}
	}
}

func OnReceive() {
}
