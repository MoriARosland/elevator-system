# Elevator-System

An elevator system with N elevators and M floors implemented in Go.

The network topology emulates a circle, where data flows from one elevator to the next.

## Usage

**IMPORTANT: Program only runs on Go ^1.21.**

The elevator system only runs on Unix-like operating systems, and requires an interface to an elevator. This can be a physical device or a simulator.

[Simulator-v2](https://github.com/TTK4145/Simulator-v2) was used in development of this system, and is the recommended simulator to run the project.

Install Simulator-v2 locally by following the readme. If you are running macOS Sonoma, you will likely encounter a linker error when compiling the simulator using the command from the readme. A quick fix can be retreived from [here](https://forum.dlang.org/thread/jwmpdecwyazcrxphttoy@forum.dlang.org).

To use Simulator-v2, see _Default keyboard controls_ from the [readme](https://github.com/TTK4145/Simulator-v2).

Each elevator instance should be paired with an elevator-server (eg. simulator). This means connecting the elevator to the port of a server.

Required flags to run the program are:

- -id: node ID of the elevator (first elevator should be 0).
- -num: number of nodes (elevators) in the network.
- -sport: which server-port the elevator should interface with.

### Example

Starting a 3-node elevator system should look something like this:

**Terminal 1:**

```bash
go run elevator -id 0 -num 3 -sport {server1-port}
```

**Terminal 2:**

```bash
go run elevator -id 1 -num 3 -sport {server2-port}
```

**Terminal 3:**

```bash
go run elevator -id 2 -num 3 -sport {server3-port}
```

## Program Notes

The program contains an elevator-state object (elevState), which serves the purpose of triggering FSM-updates at correct time with correct inputs.

Methods using the elevState will always modify it _by reference_. This means no return object is necessary. In the code however, methods frequently return a pointer to the elevState. This is purely cosmetic to highlight when the elevState is updated.

## Repository activity

![Alt](https://repobeats.axiom.co/api/embed/3cdbb9e89645f822cf0bf49fa4132340888bee60.svg "Repobeats analytics image")
