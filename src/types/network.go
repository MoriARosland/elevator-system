package types

type MsgTypes int

const (
	BID MsgTypes = iota
	ASSIGN
	REASSIGN
	SERVED
	SYNC
)

type OrderStatus int

const UNASSIGNED OrderStatus = -1

/*
 * OldAssignee is set on reassignment bids
 * and -1 for assignment bids
 */
type Bid struct {
	Order        Order
	TimeToServed []int
	OldAssignee  int
}

type Assign struct {
	Order       Order
	NewAssignee int
	OldAssignee int
}

type Served struct {
	Order Order
}

type Sync struct {
	Orders   [][][]bool
	TargetID int
}

/*
 * Header must have a fixed size
 * -> AuthorID must be btween 0 and 9
 */
type Header struct {
	AuthorID  int
	Recipient int
	UUID      string
	LoopCounter int
}

type Content interface {
	Bid | Assign | Served | Sync
}

type Msg[T Content] struct {
	Header  Header
	Content T
}
