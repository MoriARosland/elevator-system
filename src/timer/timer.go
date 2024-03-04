package timer

import "time"

var timerEndTime int64
var timerActive bool

func Start(duration int) {
	timerEndTime = time.Now().UnixMilli() + int64(duration)
	timerActive = true
}

func Stop() {
	timerActive = false
}

func TimedOut() bool {
	return timerActive && time.Now().UnixMilli() > timerEndTime
}
