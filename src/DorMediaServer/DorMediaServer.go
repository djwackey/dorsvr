package main

import (
	"flag"
	"fmt"
	"os"

	env "UsageEnvironment"
	"constant"
	media "liveMedia"
)

func main() {
	if printCommandArgs() {
		return
	}

	fmt.Println("--------------------------------")
	fmt.Println("|   Dor Media Server " + constant.MEDIA_SERVER_VERSION + "   |")
	fmt.Println("--------------------------------")

	rtspServerPortNum := 554
	rtspServer := media.NewRTSPServer(rtspServerPortNum)
	if rtspServer == nil {
		rtspServerPortNum = 8554
		rtspServer = media.NewRTSPServer(rtspServerPortNum)
	}

	if rtspServer == nil {
		fmt.Println(constant.FAILED_CREATE_SERVER)
		return
	}
	rtspServer.Start()
	fmt.Println(constant.START_MEDIA_SERVER)

	urlPrefix := rtspServer.RtspURLPrefix()
	fmt.Println("This server's URL: " + urlPrefix + "<filename>.")

	if !rtspServer.SetUpTunnelingOverHTTP(80) ||
		!rtspServer.SetUpTunnelingOverHTTP(8000) ||
		!rtspServer.SetUpTunnelingOverHTTP(8080) {
		fmt.Println(fmt.Sprintf("(We use port %d for optional RTSP-over-HTTP tunneling,"+
			"or for HTTP live streaming (for indexed Transport Stream files only).)", rtspServer.HttpServerPortNum()))
	} else {
		fmt.Println("(RTSP-over-HTTP tunneling is not available.)")
	}

	env.TaskScheduler().DoEventLoop()
	return
}

func printCommandArgs() bool {
	//daemons := false
	flag.Parse()
	if flag.NArg() >= 1 {
		switch os.Args[1] {
		case "/h", "/help":
			fmt.Println(constant.HELP_MESSAGE + "\n" + constant.HELP_DAEMON)
			break
		case "/v", "/version":
			fmt.Println(constant.MEDIA_SERVER_NAME + constant.MEDIA_SERVER_VERSION)
			break
		case "/i", "/install":
			break
		case "/u", "/uninstall":
			break
		default:
		}
		return true
	}

	return false
}
