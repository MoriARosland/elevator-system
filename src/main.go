package main

import (
	"Driver-go/elevio"
	"elevator/pack"
)

func main() {
	/*
	 * Demonstrates that the elevator driver import works
	 * Will only work when elevator server is running on specified address
	 */
	numFloors := 4
	elevServerAddr := "localhost"

	elevio.Init(elevServerAddr, numFloors)

	/*
	 * Function declared inside main package, but in diffrent file
	 */
	test()

	/*
	 * Function declared in pack (test package)
	 */
	pack.Test()
}
