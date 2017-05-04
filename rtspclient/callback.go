package rtspclient

import (
	"github.com/djwackey/dorsvr/livemedia"
	"github.com/djwackey/gitea/log"
)

func continueAfterDESCRIBE(c *RTSPClient, resultCode int, resultStr string) {
	for {
		if resultCode != 0 {
			log.Error(4, "Failed to get a SDP description: %s", resultStr)
			break
		}

		sdpDesc := resultStr

		scs := c.scs
		// Create a media session object from this SDP description
		scs.Session = livemedia.NewMediaSession(sdpDesc)
		if scs.Session == nil {
			log.Error(4, "Failed to create a MediaSession object from the sdp Description.")
			break
		} else if !scs.Session.HasSubsessions() {
			log.Error(4, "This session has no media subsessions (i.e., no \"-m\" lines)")
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
			log.Error(4, "Failed to set up the subsession")
			break
		}

		scs := c.scs
		scs.Subsession.Sink = NewDummySink(scs.Subsession, c.baseURL)
		if scs.Subsession.Sink == nil {
			log.Error(4, "Failed to create a data sink for the subsession.")
			break
		}

		log.Info("Created a data sink for the \"%s/%s\" subsession.",
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
			log.Error(4, "Failed to start playing session: %s", resultStr)
			break
		}

		log.Info("Started playing session")
		return
	}

	// An unrecoverable error occurred with this stream.
	shutdownStream(c)
}

func subsessionByeHandler(subsession *livemedia.MediaSubsession) {
	log.Info("Received RTCP BYE on subsession.")

	// Now act as if the subsession had closed:
	subsessionAfterPlaying(subsession)
}

func subsessionAfterPlaying(subsession *livemedia.MediaSubsession) {
	rtspClient := subsession.MiscPtr.(*RTSPClient)
	shutdownStream(rtspClient)
}

func shutdownStream(c *RTSPClient) {
	if c != nil {
		c.sendTeardownCommand(c.scs.Session, nil)
	}

	log.Info("Closing the Stream.")
}

func setupNextSubSession(c *RTSPClient) {
	scs := c.scs
	scs.Subsession = scs.Next()
	if scs.Subsession != nil {
		if !scs.Subsession.Initiate() {
			log.Error(4, "Failed to initiate the subsession.")
			setupNextSubSession(c)
		} else {
			log.Info("Initiated the \"%s/%s\" subsession (client ports %d-%d)",
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
