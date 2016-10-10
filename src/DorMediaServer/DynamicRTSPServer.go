package main

import (
	media "liveMedia"
)

type DynamicRTSPServer struct {
	media.RTSPServerSupportingHTTPStreaming
}

func (server *DynamicRTSPServer) InitDynamicRTSPServer() {
}

func (server *DynamicRTSPServer) LookupServerMediaSession(streamName string) *media.ServerMediaSession {
	return nil
}
