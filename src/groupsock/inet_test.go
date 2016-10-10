package groupsock

import (
	"fmt"
	"testing"
)

func Test_OurRandom32(t *testing.T) {
	sessionID := fmt.Sprintf("%010d", OurRandom32())
	//sessionID := fmt.Sprintf("%010d%010d", OurRandom32(), OurRandom32())
	if len(sessionID) == 10 {
		fmt.Println("SessionId:", sessionID)
		t.Log("success")
	} else {
		t.Error("failed")
	}
}
