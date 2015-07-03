package liveMedia

type ServerMediaSubSession struct {
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
