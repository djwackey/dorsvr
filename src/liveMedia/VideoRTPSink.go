package liveMedia

type VideoRTPSink struct {
	MultiFramedRTPSink
}

func (this *VideoRTPSink) InitVideoRTPSink() {
    this.InitMultiFramedRTPSink()
}
