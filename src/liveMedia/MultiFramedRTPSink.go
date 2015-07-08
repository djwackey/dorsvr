package liveMedia

type MultiFramedRTPSink struct {
	RTPSink
}

func (this *MultiFramedRTPSink) InitMultiFramedRTPSink() {
	//this.InitRTPSink()
}

func (this *MultiFramedRTPSink) continuePlaying() {
	this.buildAndSendPacket()
}

func (this *MultiFramedRTPSink) buildAndSendPacket() {
	this.packFrame()
}

func (this *MultiFramedRTPSink) packFrame() {
	this.afterGettingFrame()
}

func (this *MultiFramedRTPSink) afterGettingFrame() {
	this.sendPacketIfNecessary()
}

func (this *MultiFramedRTPSink) sendPacketIfNecessary() {
	//this.rtpInterface.sendPacket()
}
