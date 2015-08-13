package groupsock

import (
	"math/rand"
	"time"
)

func OurRandom32() uint {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	random16_1 := r.Int31() & 0x00FFFF00
	random16_2 := r.Int31() & 0x00FFFF00
	return uint((random16_1 << 8) | (random16_2 >> 8))
}
