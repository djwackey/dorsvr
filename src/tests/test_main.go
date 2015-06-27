package main

import (
	"fmt"
	//. "groupsock"
	//. "include"
	//. "liveMedia"
)

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
	for i := 0; i < 10; i++ {
		fmt.Println(i)
	}

	n := 10

	n++
	fmt.Println(n)
}
