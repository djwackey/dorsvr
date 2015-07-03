package liveMedia

type MultiFramedRTPSink struct {
	RTPSink
}

func (this *MultiFramedRTPSink) InitMultiFramedRTPSink() {
    this.InitRTPSink()
}

func (this *MultiFramedRTPSink) continuePlaying() {
    buildAndSendPacket()
}

func (this *MultiFramedRTPSink) buildAndSendPacket() {
    this.packFrame()
}

func (this *MultiFramedRTPSink) packFrame() {
    afterGettingFrame()
}

func (this *MultiFramedRTPSink) afterGettingFrame() {
    sendPacketIfNecessary()
}

func (this *MultiFramedRTPSink) sendPacketIfNecessary() {
    rtpInterface.sendPacket()
}
