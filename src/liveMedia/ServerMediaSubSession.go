package liveMedia

type ServerMediaSubSession interface {
	//NewStreamSource(estBitrate uint)
	//NewRTPSink()
}

//
func rangeSDPLine() string {
	return "a=range:npt=0-\r\n"
}

func getAuxSDPLine(rtpSink *RTPSink) interface{} {
	if rtpSink == nil {
		return nil
	}

	return rtpSink.AuxSDPLine()
}
