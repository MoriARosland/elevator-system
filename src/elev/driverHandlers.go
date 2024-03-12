package elev

import (
	"Driver-go/elevio"
	"elevator/fsm"
	"elevator/network"
	"elevator/types"
)

func HandleNewOrder(
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
	order types.Order,
	servedTx chan types.Msg[types.Served],
	bidTx chan types.Msg[types.Bid],
	assignTx chan types.Msg[types.Assign],
	doorTimer chan<- types.TimerActions,
	floorTimer chan<- types.TimerActions,
) *types.ElevState {

	isCabOrder := order.Button == elevio.BT_Cab

	if elevState.Disconnected && isCabOrder {
		/*
		 * When disconnected we only handle new cab orders
		 */
		elevState = SelfAssignOrder(
			elevState,
			elevConfig,
			order,
			servedTx,
			doorTimer,
			floorTimer,
		)
	} else if isCabOrder {
		/*
		 * Cab orders are selfassigned (over the network)
		 */
		assignTx <- network.FormatAssignMsg(
			order,
			elevConfig.NodeID,
			int(types.UNASSIGNED),
			elevState.NextNodeID,
			elevConfig.NodeID,
		)
	} else {
		/*
		 * Hall orders are assigned after a bidding round
		 */
		bidTx <- network.FormatBidMsg(
			nil,
			order,
			int(types.UNASSIGNED),
			elevConfig.NumNodes,
			elevState.NextNodeID,
			elevConfig.NodeID,
		)
	}

	return elevState
}

func HandleFloorArrival(
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
	newFloor int,
	servedTx chan types.Msg[types.Served],
	doorTimer chan<- types.TimerActions,
	floorTimer chan<- types.TimerActions,
) *types.ElevState {

	oldFloor := elevState.Floor

	elevState.Floor = newFloor
	elevio.SetFloorIndicator(newFloor)

	floorTimer <- types.STOP
	elevState.StuckBetweenFloors = false

	fsmOutput := fsm.OnFloorArrival(elevState, elevConfig)

	elevState = SetState(
		elevState,
		elevConfig,
		fsmOutput,
		servedTx,
		doorTimer,
		floorTimer,
	)

	if !fsmOutput.SetMotor && oldFloor != -1 {
		floorTimer <- types.START
	}

	return elevState
}

func HandleDoorTimeout(
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
	servedTx chan types.Msg[types.Served],
	doorTimer chan<- types.TimerActions,
	floorTimer chan<- types.TimerActions,
) *types.ElevState {

	if elevState.DoorObstr {
		doorTimer <- types.START
		return elevState
	}
	doorTimer <- types.STOP

	fsmOutput := fsm.OnDoorTimeout(elevState, elevConfig)

	elevState = SetState(
		elevState,
		elevConfig,
		fsmOutput,
		servedTx,
		doorTimer,
		floorTimer,
	)

	return elevState
}
