# rtspclient
It 's a RTSP client implemented by golang

## Install
```bash
go get github.com/djwackey/dorsvr/rtspclient
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
