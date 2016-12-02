package main

import (
	env "UsageEnvironment"
	"flag"
	"fmt"
	. "liveMedia"
	"os"
	"utils"
)

type ourRTSPClient struct {
	RTSPClient
}

type DummySink struct {
	MediaSink
	streamID      string
	receiveBuffer []byte
	subsession    *MediaSubSession
}

var rtspClientCount int

func NewOurRTSPClient(appName, rtspURL string) *ourRTSPClient {
	rtspClient := new(ourRTSPClient)
	rtspClient.InitRTSPClient(rtspURL, appName)
	return rtspClient
}

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		usage(os.Args[0])
		return
	}

	os.Args[0] = "dorsvr"

	if openURL(os.Args[0], os.Args[1]) {
		env.TaskScheduler().DoEventLoop()
	}
}

func usage(progName string) {
	fmt.Println("Usage: " + progName + " <rtsp-url-1> ... <rtsp-url-N>")
	fmt.Println("\t(where each <rtsp-url-i> is a \"rtsp://\" URL)")
}

func openURL(appName, rtspURL string) bool {
	rtspClient := NewOurRTSPClient(appName, rtspURL)
	if rtspClient == nil {
		fmt.Println("Failed to create a RTSP client URL", rtspURL)
		return false
	}

	rtspClientCount++

	sendBytes := rtspClient.SendDescribeCommand(continueAfterDESCRIBE)
	if sendBytes == 0 {
		fmt.Println("Failed to send describe command.")
		return false
	}

	return true
}

func continueAfterDESCRIBE(rtspClient *RTSPClient, resultCode int, resultStr string) {
	for {
		if resultCode != 0 {
			fmt.Println(fmt.Sprintf("Failed to get a SDP description: %s", resultStr))
			break
		}

		sdpDesc := resultStr

		scs := rtspClient.SCS()
		// Create a media session object from this SDP description
		scs.Session = NewMediaSession(sdpDesc)
		if scs.Session == nil {
			fmt.Println("Failed to create a MediaSession object from the sdp Description.")
			break
		} else if !scs.Session.HasSubSessions() {
			fmt.Println("This session has no media subsessions (i.e., no \"-m\" lines)")
			break
		}

		// Then, create and set up our data source objects for the session.
		setupNextSubSession(rtspClient)
		return
	}

	// An error occurred with this stream.
	shutdownStream(rtspClient)
}

func continueAfterSETUP(rtspClient *RTSPClient, resultCode int, resultStr string) {
	for {
		if resultCode != 0 {
			fmt.Println("Failed to set up the subsession")
			break
		}

		scs := rtspClient.SCS()
		scs.Subsession.Sink = NewDummySink(scs.Subsession, rtspClient.URL())
		if scs.Subsession.Sink == nil {
			fmt.Println("Failed to create a data sink for the subsession.")
			break
		}

		fmt.Printf("Created a data sink for the \"%s/%s\" subsession\n",
			scs.Subsession.MediumName(), scs.Subsession.CodecName())

		scs.Subsession.MiscPtr = rtspClient
		scs.Subsession.Sink.StartPlaying(scs.Subsession.ReadSource())
		if scs.Subsession.RtcpInstance() != nil {
			scs.Subsession.RtcpInstance().SetByeHandler(subsessionByeHandler, scs.Subsession)
		}
		break
	}

	// Set up the next subsession, if any:
	setupNextSubSession(rtspClient)
}

func continueAfterPLAY(rtspClient *RTSPClient, resultCode int, resultStr string) {
	for {
		if resultCode != 0 {
			fmt.Println(fmt.Sprintf("Failed to start playing session: %s", resultStr))
			break
		}

		fmt.Println("Started playing session")
		return
	}

	// An unrecoverable error occurred with this stream.
	shutdownStream(rtspClient)
}

func subsessionByeHandler(subsession *MediaSubSession) {
	fmt.Println("Received RTCP BYE on subsession.")

	// Now act as if the subsession had closed:
	subsessionAfterPlaying(subsession)
}

func subsessionAfterPlaying(subsession *MediaSubSession) {
	var rtspClient *RTSPClient = subsession.MiscPtr.(*RTSPClient)
	shutdownStream(rtspClient)
}

func shutdownStream(rtspClient *RTSPClient) {
	scs := rtspClient.SCS()

	//if scs.Subsession.RtcpInstance() != nil {
	//	scs.Subsession.RtcpInstance().SetByeHandler(nil, nil)
	//}

	if rtspClient != nil {
		rtspClient.SendTeardownCommand(scs.Session, nil)
	}

	fmt.Println("Closing the Stream.")

	rtspClientCount--
	if rtspClientCount == 0 {
		os.Exit(1)
	}
}

func setupNextSubSession(rtspClient *RTSPClient) {
	scs := rtspClient.SCS()
	scs.Subsession = scs.Next()
	if scs.Subsession != nil {
		if !scs.Subsession.Initiate() {
			fmt.Println("Failed to initiate the subsession.")
			setupNextSubSession(rtspClient)
		} else {
			fmt.Printf("Initiated the \"%s/%s\" subsession (client ports %d-%d)\n",
				scs.Subsession.MediumName(), scs.Subsession.CodecName(),
				scs.Subsession.ClientPortNum(), scs.Subsession.ClientPortNum()+1)
			rtspClient.SendSetupCommand(scs.Subsession, continueAfterSETUP)
		}
		return
	}

	if scs.Session.AbsStartTime() != "" {
		rtspClient.SendPlayCommand(scs.Session, continueAfterPLAY)
	} else {
		rtspClient.SendPlayCommand(scs.Session, continueAfterPLAY)
	}
}

// Implementation of "DummySink":

var dummySinkReceiveBufferSize uint = 100000

func NewDummySink(subsession *MediaSubSession, streamID string) *DummySink {
	sink := new(DummySink)
	sink.streamID = streamID
	sink.subsession = subsession
	sink.receiveBuffer = make([]byte, dummySinkReceiveBufferSize)
	sink.InitMediaSink(sink)
	return sink
}

func (sink *DummySink) AfterGettingFrame(frameSize, durationInMicroseconds uint,
	presentationTime utils.Timeval) {
	fmt.Printf("Stream \"%s\"; %s/%s:\tReceived %d bytes.\tPresentation Time: %f\n",
		sink.streamID, sink.subsession.MediumName(), sink.subsession.CodecName(), frameSize,
		float32(presentationTime.Tv_sec/1000/1000+presentationTime.Tv_usec))
}

func (sink *DummySink) ContinuePlaying() {
	sink.Source.GetNextFrame(sink.receiveBuffer, dummySinkReceiveBufferSize,
		sink.AfterGettingFrame, sink.OnSourceClosure)
}
