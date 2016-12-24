package utils

import "time"

type Timeval struct {
	Tv_sec  int64
	Tv_usec int64
}

func GetTimeOfDay(val *Timeval) {
	tv_nsec := time.Now().UnixNano()
	val.Tv_sec = tv_nsec / 1000000000
	val.Tv_usec = tv_nsec % (val.Tv_sec * 1000000000)
}
