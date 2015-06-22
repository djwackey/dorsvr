package main

import (
	"fmt"
	. "groupsock"
)

func main() {
	sessionId := OurRandom32()
	fmt.Println(sessionId)
}
