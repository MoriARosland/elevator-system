package fsm

import (
	"Driver-go/elevio"
	"elevator/orders"
	"elevator/types"
	"encoding/json"
)

const TRAVEL_TIME = 2000 // ms

func deepCopy(obj types.ElevState, copy *types.ElevState) {
	encodedObj, _ := json.Marshal(obj)
	_ = json.Unmarshal(encodedObj, copy)
}

func TimeToOrderServed(elevState *types.ElevState, elevConfig *types.ElevConfig, order types.Order) int {
	if 0 > elevState.Floor {
		return -1
	}

	var elevSimState types.ElevState
	deepCopy(*elevState, &elevSimState)

	elevSimState.Orders[elevConfig.NodeID][order.Floor][order.Button] = true

	duration := 0

	switch state {
	case types.EB_Idle:
		elevSimState.Dirn = orders.ChooseDirection(&elevSimState, elevConfig).Dirn

		if elevSimState.Dirn == elevio.MD_Stop {
			return duration
		}

	case types.EB_Moving:
		duration += TRAVEL_TIME / 2
		elevSimState.Floor += int(elevSimState.Dirn)

	case types.EB_DoorOpen:
		duration -= elevConfig.DoorOpenDuration / 2
	}

	for {
		if orders.ShouldStop(&elevSimState, elevConfig) {
			shouldClear := orders.ClearAtCurrentFloor(&elevSimState, elevConfig)

			if order.Floor == elevSimState.Floor && shouldClear[order.Button] {
				if 0 > duration {
					duration = 0
				}
				return duration
			}

			// TODO: Add clear order func
			for btn, clearButton := range shouldClear {
				if clearButton {
					elevSimState.Orders[elevConfig.NodeID][elevSimState.Floor][btn] = false
				}
			}

			duration += elevConfig.DoorOpenDuration
			elevSimState.Dirn = orders.ChooseDirection(&elevSimState, elevConfig).Dirn
		}

		elevSimState.Floor += int(elevSimState.Dirn)
		duration += TRAVEL_TIME
	}
}
