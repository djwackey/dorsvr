package liveMedia

// allow for some possibly large H.264 frames
var maxSize uint = 100000

type MediaSink struct {
}

type OutPacketBuffer struct {
}

func NewOutPacketBuffer() *OutPacketBuffer {
    return &OutPacketBuffer{}
}

func (this *MediaSink) startPlaying() {
}

func (this *MediaSink) stopPlaying() {
}

