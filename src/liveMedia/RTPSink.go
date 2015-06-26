package liveMedia

type RTPSink struct {
    MediaSink
}

func (this *RTPSink) auxSDPLine() {
    return nil
}
