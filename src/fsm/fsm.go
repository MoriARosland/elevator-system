package fsm

import (
	"Driver-go/elevio"
	"elevator/elev"
)

var State ElevBehaviour = EB_Idle

func OnInitBetweenFloors() {
	State = EB_Moving
}

func OnRequestButtonPress(
	buttonPress elevio.ButtonEvent,
	elevState *elev.ElevState,
) {

}

func OnFloorArrival(
	floor int,
	elevState *elev.ElevState,
) {

}
