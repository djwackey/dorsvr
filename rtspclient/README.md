# rtspclient
It 's a RTSP client implemented by golang

[![Build Status](https://travis-ci.org/djwackey/rtspclient.svg?branch=master)](https://travis-ci.org/djwackey/rtspclient) [![GitHub issues](https://img.shields.io/github/issues/djwackey/rtspclient.svg)](https://github.com/djwackey/rtspclient/issues)

## Install

```bash
go get github.com/djwackey/rtspclient
```

## Examples

```go
// define a rtsp url
rtsp_url := "rtsp://192.168.1.105:8554/demo.264"

// create a rtspclient instance
client := rtspclient.New()

// connect rtsp server and send request
client.DialRTSP(rtsp_url)

// waiting for the response, and print the output frame data
client.Waiting()

```
