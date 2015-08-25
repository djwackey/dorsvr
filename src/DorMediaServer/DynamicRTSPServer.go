package main

import (
	. "liveMedia"
)

type DynamicRTSPServer struct {
	RTSPServerSupportingHTTPStreaming
}

func (this *DynamicRTSPServer) InitDynamicRTSPServer() {
}

func (this *DynamicRTSPServer) LookupServerMediaSession(streamName string) *ServerMediaSession {
	return nil
}
