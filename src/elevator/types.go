package elevator

type Next struct {
	ID   int
	Addr string
}

type Elevator struct {
	NodeID        int
	NumNodes      int
	BroadCastPort int
	Next          Next
}
