package rtspclient

import (
	"fmt"
	"github.com/djwackey/dorsvr/livemedia"
	"github.com/djwackey/dorsvr/utils"
)

type DummySink struct {
	livemedia.MediaSink
	streamID      string
	receiveBuffer []byte
	subsession    *livemedia.MediaSubSession
}

// Implementation of "DummySink":

var dummySinkReceiveBufferSize uint = 100000

func NewDummySink(subsession *livemedia.MediaSubSession, streamID string) *DummySink {
	sink := new(DummySink)
	sink.streamID = streamID
	sink.subsession = subsession
	sink.receiveBuffer = make([]byte, dummySinkReceiveBufferSize)
	sink.InitMediaSink(sink)
	return sink
}

func (sink *DummySink) AfterGettingFrame(frameSize, durationInMicroseconds uint,
	presentationTime utils.Timeval) {
	//return
	fmt.Printf("Stream \"%s\"; %s/%s:\tReceived %d bytes.\tPresentation Time: %f\n",
		sink.streamID, sink.subsession.MediumName(), sink.subsession.CodecName(), frameSize,
		float32(presentationTime.Tv_sec/1000/1000+presentationTime.Tv_usec))
}

func (sink *DummySink) ContinuePlaying() {
	sink.Source.GetNextFrame(sink.receiveBuffer, dummySinkReceiveBufferSize,
		sink.AfterGettingFrame, sink.OnSourceClosure)
}
