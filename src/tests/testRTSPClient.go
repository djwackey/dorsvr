package main

import (
	env "UsageEnvironment"
	"flag"
	"fmt"
	. "liveMedia"
	"os"
)

type OurRTSPClient struct {
	RTSPClient
}

var rtspClientCount int

func NewOurRTSPClient(appName, rtspURL string) *OurRTSPClient {
	rtspClient := new(OurRTSPClient)
	rtspClient.InitRTSPClient(rtspURL, appName)
	return rtspClient
}

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		usage(os.Args[0])
		return
	}

	os.Args[0] = "dorsvr"

	if openURL(os.Args[0], os.Args[1]) {
		env.TaskScheduler().DoEventLoop()
	}
}

func usage(progName string) {
	fmt.Println("Usage: " + progName + " <rtsp-url-1> ... <rtsp-url-N>")
	fmt.Println("\t(where each <rtsp-url-i> is a \"rtsp://\" URL)")
}

func openURL(appName, rtspURL string) bool {
	rtspClient := NewOurRTSPClient(appName, rtspURL)
	if rtspClient == nil {
		fmt.Println("Failed to create a RTSP client URL", rtspURL)
		return false
	}

	rtspClientCount++

	sendBytes := rtspClient.SendDescribeCommand(continueAfterDESCRIBE)
	fmt.Println("sendBytes: ", sendBytes)
	if sendBytes == 0 {
		return false
	}

	return true
}

func continueAfterDESCRIBE() {
	fmt.Println("continueAfterDESCRIBE")
}

func continueAfterSETUP() {
}

func continueAfterPLAY() {
}

func setupNextSubSession(rtspClient *RTSPClient) {
	rtspClient.SendPlayCommand(continueAfterPLAY)
}
