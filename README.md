# Elevator-System

An elevator system with N elevators and M floors implemented in Go.

The network topology of the system is circular, where data flows from one elevator to the next.

## Program Notes

**IMPORTANT: Program only runs on Go ^1.21.**

The program contains an elevator-state object (elevState), which serves the purpose of triggering FSM-updates at correct time with correct inputs.

Methods using the elevState will always modify it _by reference_. This means no return object is necessary. In the code however, methods frequently return a pointer to the elevState. This is purely cosmetic to highlight when the elevState is updated.

## Installation

The elevator system only runs on Unix-like operating systems, and requires an interface to an elevator. This can be a physical device or a simulator.

[Simulator-v2](https://github.com/TTK4145/Simulator-v2) was used in development of this system, and is the recommended simulator to run the project.

Install Simulator-v2 locally and follow the readme. If you are running macOS Sonoma, you will likely encounter a linker error when compiling the simulator using the command from the readme. A quick fix can be retreived from [here](https://forum.dlang.org/thread/jwmpdecwyazcrxphttoy@forum.dlang.org).

Clone the repository, cd to the project, and use `go build elevator` from the terminal to build the program.

## Usage

Each elevator instance should be paired with an elevator-server (eg. simulator). This means connecting the elevator to the port of a server.

Open the terminal from the root of the project, and enter `elevator -h`. This command lists the required flags to start an elevator-instance. The flags are:

- -id: node ID of the elevator (first elevator should be 0).
- -num: number of nodes (elevators) in the network.
- -sport: which server-port the elevator should interface with.

### Example 1: Starting a single elevator

1. From the root of the project, build the program with: `go build elevator`.
2. Start a elevator-server (eg. simulator) and note which port to connect to.
3. Start a single elevator instance with: `elevator -id 0 -num 1 -sport 8080`.

### Example 2: Starting multiple elevators

Repeat the steps from example 1, but increment -id and set corresponding -sport for each new elevator instance.

Starting a 3-node elevator system should look something like this:

**Terminal 1:**

```bash
elevator -num 3 -id 0 -sport {server1-port}
```

**Terminal 2:**

```bash
elevator -num 3 -id 1 -sport {server2-port}
```

**Terminal 3:**

```bash
elevator -num 3 -id 2 -sport {server3-port}
```

To use the Simulator-v2, see the [readme](https://github.com/TTK4145/Simulator-v2).

## Repository activity

![Alt](https://repobeats.axiom.co/api/embed/3cdbb9e89645f822cf0bf49fa4132340888bee60.svg "Repobeats analytics image")
