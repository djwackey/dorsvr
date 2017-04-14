package rtspclient

import (
	"fmt"
	sys "syscall"

	"github.com/djwackey/dorsvr/livemedia"
)

type DummySink struct {
	livemedia.MediaSink
	streamID      string
	receiveBuffer []byte
	subsession    *livemedia.MediaSubsession
}

// Implementation of "DummySink":

var dummySinkReceiveBufferSize uint = 100000

func NewDummySink(subsession *livemedia.MediaSubsession, streamID string) *DummySink {
	sink := new(DummySink)
	sink.streamID = streamID
	sink.subsession = subsession
	sink.receiveBuffer = make([]byte, dummySinkReceiveBufferSize)
	sink.InitMediaSink(sink)
	return sink
}

func (s *DummySink) AfterGettingFrame(frameSize, durationInMicroseconds uint,
	presentationTime sys.Timeval) {
	//return
	fmt.Printf("Stream \"%s\"; %s/%s:\tReceived %d bytes.\tPresentation Time: %f\n",
		s.streamID, s.subsession.MediumName(), s.subsession.CodecName(), frameSize,
		float32(presentationTime.Sec/1000/1000+presentationTime.Usec))
}

func (s *DummySink) ContinuePlaying() {
	s.Source.GetNextFrame(s.receiveBuffer, dummySinkReceiveBufferSize,
		s.AfterGettingFrame, s.OnSourceClosure)
}
