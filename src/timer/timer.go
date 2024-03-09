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
