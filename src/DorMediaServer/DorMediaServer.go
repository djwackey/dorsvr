package main

import (
	"flag"
	"fmt"
	"os"

	. "DorDatabase"
	env "UsageEnvironment"
	. "include"
	. "liveMedia"
)

func main() {
	if PrintCommandArgs() {
		return
	}

	fmt.Println("--------------------------------")
	fmt.Println("|   Dor Media Server " + MEDIA_SERVER_VERSION + "   |")
	fmt.Println("--------------------------------")

	// create an instance of configure file manager.
	confFileManager := NewConfFileManager()
	if confFileManager == nil {
		return
	}

	if !confFileManager.ReadConfInfo(DORMS_CONFIG_FILE) {
		fmt.Println(FAILED_READ_CONFIG)
		return
	}
	fmt.Println(SUCCESS_READ_CONFIG)

	rtspServerPortNum := 554
	rtspServer := NewRTSPServer(rtspServerPortNum)
	if rtspServer == nil {
		rtspServerPortNum = 8554
		rtspServer = NewRTSPServer(rtspServerPortNum)
	}

	if rtspServer == nil {
		fmt.Println(FAILED_CREATE_SERVER)
		return
	}
	rtspServer.Start()
	fmt.Println(START_MEDIA_SERVER)

	urlPrefix := rtspServer.RtspURLPrefix()
	fmt.Println("This server's URL: " + urlPrefix + "<filename>.")

    if !rtspServer.SetUpTunnelingOverHTTP(80) || !rtspServer.SetUpTunnelingOverHTTP(8000) || !rtspServer.SetUpTunnelingOverHTTP(8080) {
        fmt.Println(fmt.Sprintf("(We use port %d for optional RTSP-over-HTTP tunneling, or for HTTP live streaming (for indexed Transport Stream files only).)", rtspServer.HttpServerPortNum()))
    } else {
        fmt.Println("(RTSP-over-HTTP tunneling is not available.)")
    }

	env.TaskScheduler().DoEventLoop()
	return
}

func PrintCommandArgs() bool {
	//daemons := false
	flag.Parse()
	if flag.NArg() >= 1 {
		switch os.Args[1] {
		case "/h", "/help":
			fmt.Println(HELP_MESSAGE + "\n" + HELP_DAEMON)
			break
		case "/v", "/version":
			fmt.Println(MEDIA_SERVER_NAME + MEDIA_SERVER_VERSION)
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
