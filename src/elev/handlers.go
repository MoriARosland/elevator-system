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
	syncTx chan types.Msg[types.Sync],
	bidTx chan types.Msg[types.Bid],
	assignTx chan types.Msg[types.Assign],
	doorTimer chan<- types.TimerActions,
	floorTimer chan<- types.TimerActions,
) *types.ElevState {

	isCabOrder := order.Button == elevio.BT_Cab
	isAlone := elevState.NextNodeID == elevConfig.NodeID

	if isAlone && isCabOrder {
		elevState = SelfAssignOrder(
			elevState,
			elevConfig,
			order,
			servedTx,
			syncTx,
			doorTimer,
			floorTimer,
		)
	} else if !isAlone && isCabOrder {
		assignTx <- network.FormatAssignMsg(
			order,
			elevConfig.NodeID,
			int(types.UNASSIGNED),
			elevState.NextNodeID,
			elevConfig.NodeID,
		)
	} else {
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

func HandleDoorObstr(
	elevState *types.ElevState,
	isObstructed bool,
	obstrTimer chan types.TimerActions,
	doorTimer chan types.TimerActions,
) *types.ElevState {

	if elevState.DoorObstr == isObstructed {
		return elevState
	}

	if isObstructed {
		obstrTimer <- types.START
	} else {
		obstrTimer <- types.STOP
	}

	doorTimer <- types.START
	elevState.DoorObstr = isObstructed

	return elevState
}

func HandleFloorArrival(
	elevState *types.ElevState,
	elevConfig *types.ElevConfig,
	newFloor int,
	servedTx chan types.Msg[types.Served],
	syncTx chan types.Msg[types.Sync],
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
		syncTx,
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
	syncTx chan types.Msg[types.Sync],
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
		syncTx,
		doorTimer,
		floorTimer,
	)

	return elevState
}