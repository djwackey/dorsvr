package liveMedia

// allow for some possibly large H.264 frames
var maxSize uint = 100000

type MediaSink struct {
}

func (this *MediaSink) InitMediaSink() {
}

type OutPacketBuffer struct {
}

func NewOutPacketBuffer() *OutPacketBuffer {
	return new(OutPacketBuffer)
}

func (this *MediaSink) startPlaying() {
}

func (this *MediaSink) stopPlaying() {
}
