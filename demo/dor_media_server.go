package main

import (
	"fmt"

	"github.com/djwackey/dorsvr/rtspserver"
	"github.com/djwackey/dorsvr/scheduler"
)

func main() {
	server := rtspserver.New()

	portNum := 8554
	server.Listen(portNum)

	if !server.SetUpTunnelingOverHTTP(80) ||
		!server.SetUpTunnelingOverHTTP(8000) ||
		!server.SetUpTunnelingOverHTTP(8080) {
		fmt.Println(fmt.Sprintf("(We use port %d for optional RTSP-over-HTTP tunneling, "+
			"or for HTTP live streaming (for indexed Transport Stream files only).)",
			server.HttpServerPortNum()))
	} else {
		fmt.Println("(RTSP-over-HTTP tunneling is not available.)")
	}

	urlPrefix := server.RtspURLPrefix()
	fmt.Println("This server's URL: " + urlPrefix + "<filename>.")

	server.Start()

	scheduler.DoEventLoop()
}
