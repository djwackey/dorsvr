package rtspclient

import (
	"fmt"

	"github.com/djwackey/dorsvr/livemedia"
)

func continueAfterDESCRIBE(c *RTSPClient, resultCode int, resultStr string) {
	for {
		if resultCode != 0 {
			fmt.Printf("Failed to get a SDP description: %s\n", resultStr)
			break
		}

		sdpDesc := resultStr

		scs := c.scs
		// Create a media session object from this SDP description
		scs.Session = livemedia.NewMediaSession(sdpDesc)
		if scs.Session == nil {
			fmt.Println("Failed to create a MediaSession object from the sdp Description.")
			break
		} else if !scs.Session.HasSubSessions() {
			fmt.Println("This session has no media subsessions (i.e., no \"-m\" lines)")
			break
		}

		// Then, create and set up our data source objects for the session.
		setupNextSubSession(c)
		return
	}

	// An error occurred with this stream.
	shutdownStream(c)
}

func continueAfterSETUP(c *RTSPClient, resultCode int, resultStr string) {
	for {
		if resultCode != 0 {
			fmt.Println("Failed to set up the subsession")
			break
		}

		scs := c.scs
		scs.Subsession.Sink = NewDummySink(scs.Subsession, c.baseURL)
		if scs.Subsession.Sink == nil {
			fmt.Println("Failed to create a data sink for the subsession.")
			break
		}

		fmt.Printf("Created a data sink for the \"%s/%s\" subsession\n",
			scs.Subsession.MediumName(), scs.Subsession.CodecName())

		scs.Subsession.MiscPtr = c
		scs.Subsession.Sink.StartPlaying(scs.Subsession.ReadSource(), nil)
		if scs.Subsession.RtcpInstance() != nil {
			scs.Subsession.RtcpInstance().SetByeHandler(subsessionByeHandler, scs.Subsession)
		}
		break
	}

	// Set up the next subsession, if any:
	setupNextSubSession(c)
}

func continueAfterPLAY(c *RTSPClient, resultCode int, resultStr string) {
	for {
		if resultCode != 0 {
			fmt.Printf("Failed to start playing session: %s\n", resultStr)
			break
		}

		fmt.Println("Started playing session")
		return
	}

	// An unrecoverable error occurred with this stream.
	shutdownStream(c)
}

func subsessionByeHandler(subsession *livemedia.MediaSubSession) {
	fmt.Println("Received RTCP BYE on subsession.")

	// Now act as if the subsession had closed:
	subsessionAfterPlaying(subsession)
}

func subsessionAfterPlaying(subsession *livemedia.MediaSubSession) {
	rtspClient := subsession.MiscPtr.(*RTSPClient)
	shutdownStream(rtspClient)
}

func shutdownStream(c *RTSPClient) {
	if c != nil {
		c.sendTeardownCommand(c.scs.Session, nil)
	}

	fmt.Println("Closing the Stream.")
}

func setupNextSubSession(c *RTSPClient) {
	scs := c.scs
	scs.Subsession = scs.Next()
	if scs.Subsession != nil {
		if !scs.Subsession.Initiate() {
			fmt.Println("Failed to initiate the subsession.")
			setupNextSubSession(c)
		} else {
			fmt.Printf("Initiated the \"%s/%s\" subsession (client ports %d-%d)\n",
				scs.Subsession.MediumName(), scs.Subsession.CodecName(),
				scs.Subsession.ClientPortNum(), scs.Subsession.ClientPortNum()+1)
			c.sendSetupCommand(scs.Subsession, continueAfterSETUP)
		}
		return
	}

	if scs.Session.AbsStartTime() != "" {
		c.sendPlayCommand(scs.Session, continueAfterPLAY)
	} else {
		c.sendPlayCommand(scs.Session, continueAfterPLAY)
	}
}
