package main

import (
	"flag"
	"fmt"
	. "liveMedia"
	"os"
	//env "UsageEnvironment"
)

type OurRTSPClient struct {
	RTSPClient
}

var rtspClientCount int

func NewOurRTSPClient(rtspURL string) *OurRTSPClient {
	rtspClient := &OurRTSPClient{}
	rtspClient.SetBaseURL(rtspURL)
	return rtspClient
}

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		usage(os.Args[0])
		return
	}

	openURL(os.Args[1])

	//env.TaskScheduler().DoEventLoop()
	select {}
}

func usage(progName string) {
	fmt.Println("Usage: " + progName + " <rtsp-url-1> ... <rtsp-url-N>")
	fmt.Println("\t(where each <rtsp-url-i> is a \"rtsp://\" URL)")
}

func openURL(rtspURL string) {
	rtspClient := NewOurRTSPClient(rtspURL)
	if rtspClient == nil {
		fmt.Println("Failed to create a RTSP client URL", rtspURL)
	}

	rtspClientCount++

	rtspClient.SendDescribeCommand(continueAfterDESCRIBE)
}

func continueAfterDESCRIBE() {
}
