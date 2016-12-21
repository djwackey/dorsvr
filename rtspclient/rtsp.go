package rtspclient

import (
	"fmt"
)

func New() *RTSPClient {
	return new(RTSPClient)
}

func (client *RTSPClient) DialRTSP(rtspURL string) bool {
	appName := "dorcli"
	client.InitRTSPClient(rtspURL, appName)

	sendBytes := client.SendDescribeCommand(continueAfterDESCRIBE)
	if sendBytes == 0 {
		fmt.Println("Failed to send describe command.")
		return false
	}

	return true
}

func (client *RTSPClient) Waiting() {
	//TaskScheduler().DoEventLoop()
}

func (client *RTSPClient) Close() {
	scs := client.SCS()

	//if scs.Subsession.RtcpInstance() != nil {
	//	scs.Subsession.RtcpInstance().SetByeHandler(nil, nil)
	//}

	client.SendTeardownCommand(scs.Session, nil)
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
