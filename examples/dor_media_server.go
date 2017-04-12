package main

import (
	"fmt"

	"github.com/djwackey/dorsvr/rtspserver"
	"github.com/djwackey/gitea/log"
)

func main() {
	// open a logger writer of console or file mode.
	mode := "console"
	config := `{"level":3,"filename":"test.log"}`
	log.NewLogger(0, mode, config)

	// create a rtsp server
	server := rtspserver.New()

	portNum := 8554
	err := server.Listen(portNum)
	if err != nil {
		fmt.Printf("Failed to bind port: %d\n", portNum)
		return
	}

	if !server.SetupTunnelingOverHTTP(80) ||
		!server.SetupTunnelingOverHTTP(8000) ||
		!server.SetupTunnelingOverHTTP(8080) {
		fmt.Printf("We use port %d for optional RTSP-over-HTTP tunneling, "+
			"or for HTTP live streaming (for indexed Transport Stream files only).\n",
			server.HttpServerPortNum())
	} else {
		fmt.Println("(RTSP-over-HTTP tunneling is not available.)")
	}

	urlPrefix := server.RtspURLPrefix()
	fmt.Println("This server's URL: " + urlPrefix + "<filename>.")

	server.Start()

	// do event loop
	select {}
}
