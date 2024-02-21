package elevator

type NextNode struct {
	ID   int
	Addr string
}

type Elevator struct {
	NodeID        int
	NumNodes      int
	BroadCastPort int
	NextNode          NextNode
}
