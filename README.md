Dorsvr Streaming Server
=======================

[![Build Status](https://travis-ci.org/djwackey/dorsvr.svg?branch=master)](https://travis-ci.org/djwackey/dorsvr) [![GitHub issues](https://img.shields.io/github/issues/djwackey/dorsvr.svg)](https://github.com/djwackey/dorsvr/issues)
## Modules
 * rtspserver - rtsp server
 * rtspclient - rtsp client
 * groupsock  - group socket
 * scheduler  - task scheduler
 * livemedia  - media library

## Feature
 * Streaming Video (H264)
 * Streaming Audio (MP3)
 * Protocols: RTP, RTCP, RTSP

## Install
    go get github.com/djwackey/dorsvr

## Format
    $ make fmt

## Testing
    $ make test

## Example
```golang
server := rtspserver.New()

portNum := 8554
server.Listen(portNum)

if !server.SetUpTunnelingOverHTTP(80) ||
    !server.SetUpTunnelingOverHTTP(8000) ||
    !server.SetUpTunnelingOverHTTP(8080) {
    fmt.Println(fmt.Sprintf("(We use port %d for optional RTSP-over-HTTP tunneling, "+
                            "or for HTTP live streaming (for indexed Transport Stream files only).)", server.HttpServerPortNum()))
} else {
    fmt.Println("(RTSP-over-HTTP tunneling is not available.)")
}

urlPrefix := server.RtspURLPrefix()
fmt.Println("This server's URL: " + urlPrefix + "<filename>.")

server.Start()

scheduler.DoEventLoop()
```
## Author
djwackey, worcy_kiddy@126.com

## LICENSE
dorsvr is licensed under the GNU Lesser General Public License, Version 2.1. See [LICENSE](https://github.com/djwackey/dorsvr/blob/master/LICENSE) for the full license text.
