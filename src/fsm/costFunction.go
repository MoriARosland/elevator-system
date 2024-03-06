package fsm

import (
	"Driver-go/elevio"
	"elevator/requests"
	"elevator/types"
)

const TRAVEL_TIME = 2000 // ms

func TimeToOrderServed(elevState *types.ElevState, elevConfig *types.ElevConfig, request types.Order) int {
	elevSimState := *elevState
	elevSimState.Requests[elevConfig.NodeID][request.Floor][request.Button] = true

	simulationState := state

	duration := 0

	switch simulationState {
	case types.EB_Idle:
		elevState.Dirn = requests.ChooseDirection(elevState, elevConfig).Dirn

		if elevState.Dirn == elevio.MD_Stop {
			return duration
		}

	case types.EB_Moving:
		duration += TRAVEL_TIME / 2
		elevSimState.Floor += int(elevSimState.Dirn)

	case types.EB_DoorOpen:
		duration -= elevConfig.DoorOpenDuration / 2
	}

	for {
		if requests.ShouldStop(&elevSimState, elevConfig) {
			shouldClear := requests.ClearAtCurrentFloor(&elevSimState, elevConfig)

			if request.Floor == elevSimState.Floor && shouldClear[request.Button] {
				return duration
			}

			// TODO: Add clear order func
			for btn, clearButton := range shouldClear {
				if clearButton {
					elevSimState.Requests[elevConfig.NodeID][elevSimState.Floor][btn] = false
				}
			}

			duration += elevConfig.DoorOpenDuration
			elevState.Dirn = requests.ChooseDirection(elevState, elevConfig).Dirn
		}

		elevSimState.Floor += int(elevSimState.Dirn)
		duration += TRAVEL_TIME
	}
}
