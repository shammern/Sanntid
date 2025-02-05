package main

import (
	//"fmt"
	"barebone/pkg/elevator"
	"time"
)

func main() {
	// Opprett kanaler
	newOrders := make(chan elevator.Order)
	elevatorReady := make(chan bool)
	elevatorOrders := make(chan elevator.Order)

	// Start QueueManager i egen goroutine
	go elevator.QueueManager(newOrders, elevatorReady, elevatorOrders)

	// Opprett Elevator-instans (FSM)
	elevatorFSM := elevator.NewElevator(elevatorOrders, elevatorReady)

	// Start ElevatorFSM i egen goroutine
	go elevatorFSM.Run()

	// I utgangspunktet er heisen "ledig"
	go func() {
		elevatorReady <- true
	}()

	// Send noen testbestillinger
	go func() {
		for i := 1; i <= 3; i++ {
			newOrders <- elevator.Order{
				ID:    i,
				Floor: i * 2, // 2, 4, 6
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()

	// Gi litt tid for at programmet skal kjøre
	time.Sleep(15 * time.Second)
}
