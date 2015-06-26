package include

import (
    "time"
)

type timeval struct {
    tv_sec int64
    tv_usec int64
}

func gettimeofday(val *timeval) {
    tv_nsec := time.Now().UnixNano()
    val.tv_sec =  tv_nsec / 1000000000
    val.tv_usec = tv_nsec % (val.tv_sec * 1000000000)
}
