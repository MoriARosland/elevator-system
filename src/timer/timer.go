package timer

import (
	"elevator/types"
	"time"
)

func Timer(
	duration time.Duration,
	timeOut chan<- bool,
	action <-chan types.TimerActions,
) {

	timer := time.NewTimer(duration)
	timer.Stop()

	for {
		select {
		/*
		 * STOP and START timer
		 */
		case newAction := <-action:
			switch newAction {
			case types.START:
				timer.Reset(duration)

			case types.STOP:
				timer.Stop()
			}

		/*
		 * Timer timed out
		 */
		case <-timer.C:
			timeOut <- true

		default:
			continue
		}
	}
}

func New(duration time.Duration) (chan bool, chan types.TimerActions) {
	timeout := make(chan bool)
	timer := make(chan types.TimerActions)

	go Timer(
		duration*time.Millisecond,
		timeout,
		timer,
	)

	return timeout, timer
}
