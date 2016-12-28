package main

import (
	"flag"
	"fmt"
	"github.com/djwackey/dorsvr/rtspclient"
	"os"
	"time"
)

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Println("Please input rtsp url.")
		return
	}

	rtsp_url := os.Args[1]

	client := rtspclient.New()

	// to connect rtsp server
	if !client.DialRTSP(rtsp_url) {
		return
	}

	// send the options/describe request
	client.SendRequest()

	//go TimeCloser(client)
	client.Waiting()

	fmt.Println("exit")
}

func TimeCloser(client *rtspclient.RTSPClient) {
	time.Sleep(3 * time.Second)
	client.Close()
}
