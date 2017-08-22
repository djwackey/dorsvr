package main

import (
	"fmt"

	//"github.com/djwackey/dorsvr/auth"
	"github.com/djwackey/dorsvr/rtspserver"
	"github.com/djwackey/gitea/log"
)

func main() {
	// open a logger writer of console or file mode.
	mode := "console"
	config := `{"level":1,"filename":"test.log"}`
	log.NewLogger(0, mode, config)

	// to implement client access control to the RTSP server, do the following:
	// var realm string
	// authdb = auth.NewAuthDatabase(realm)
	// authdb.InsertUserRecord("username1", "password1")
	// repeat the above with each <username>, <password> that you wish to allow
	// access to the server.

	// create a rtsp server
	server := rtspserver.New(nil)

	portNum := 8554
	err := server.Listen(portNum)
	if err != nil {
		fmt.Printf("Failed to bind port: %d\n", portNum)
		return
	}

	// also, attempt to create a HTTP server for RTSP-over-HTTP tunneling.
	// Try first with the default HTTP port (80), and then with the alternative HTTP
	// port numbers (8000 and 8080).
	if !server.SetupTunnelingOverHTTP(80) ||
		!server.SetupTunnelingOverHTTP(8000) ||
		!server.SetupTunnelingOverHTTP(8080) {
		fmt.Printf("We use port %d for optional RTSP-over-HTTP tunneling, "+
			"or for HTTP live streaming (for indexed Transport Stream files only).\n",
			server.HTTPServerPortNum())
	} else {
		fmt.Println("(RTSP-over-HTTP tunneling is not available.)")
	}

	urlPrefix := server.RtspURLPrefix()
	fmt.Println("This server's URL: " + urlPrefix + "<filename>.")

	server.Start()

	// do event loop
	select {}
}
