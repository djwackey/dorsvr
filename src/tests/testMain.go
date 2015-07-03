package main

import (
	"fmt"
    "time"
	//. "groupsock"
	//. "include"
	//. "liveMedia"
)

var counter int

func livenessTimeoutTask(millisec time.Duration) {
    tm := time.NewTimer(time.Millisecond * millisec)

    for {
        select {
        case <-tm.C:
            tm.Reset(time.Millisecond * millisec)
            counter ++
            fmt.Println("livenessTimeoutTask", counter)
            if counter > 10 {
                tm.Stop()
            }
        }
    }
}

func test() {
    for {
        time.Sleep(time.Second * 3)
        fmt.Println("test")
    }
}

func main() {
	//sessionId := OurRandom32()
	//fmt.Println(sessionId)
	/*
	   var val timeval
	   gettimeofday(&val)

	   fmt.Println(val.tv_sec)
	   fmt.Println(val.tv_usec)

	   session := NewServerMediaSession("yanfei", "264")
	   fmt.Println(session.GenerateSDPDescription())
	*/
    /*
	for i := 0; i < 10; i++ {
		fmt.Println(i)
	}

	n := 10

	n++
	fmt.Println(n)
*/
    go livenessTimeoutTask(1000)
    go test()
    select{}
}
