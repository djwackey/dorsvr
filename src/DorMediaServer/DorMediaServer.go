package main

import (
	"flag"
	"fmt"
	"os"

	. "DorDatabase"
	env "UsageEnvironment"
	. "include"
)

func main() {
	if PrintCommandArgs() {
		return
	}

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
		fmt.Println("Failed to create RTSP server.")
		return
	}
	rtspServer.Start()

	fmt.Println("Start Digital Operation Room Media Server.Version(1.0.0.3)")

	urlPrefix := rtspServer.RtspURLPrefix()
	fmt.Println("This server's URL: " + urlPrefix + "<filename>.")

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
