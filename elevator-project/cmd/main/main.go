package main

import (
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/elevator"
	"elevator-project/pkg/orders"
)

func main() {
	numFloors := 4

	drivers.Init("localhost:15657", numFloors)

	// Create the request matrix.
	rm := orders.NewRequestMatrix(numFloors)

	// Instantiate the elevator FSM with the request matrix and initial floor.
	elevatorFSM := elevator.NewElevator(rm)
	go elevatorFSM.Run()

	// Create channels to receive events from the drivers.
	drvButtons := make(chan drivers.ButtonEvent)
	drvFloors := make(chan int)
	drvObstr := make(chan bool)
	drvStop := make(chan bool)

	go drivers.PollButtons(drvButtons)
	go drivers.PollFloorSensor(drvFloors)
	go drivers.PollObstructionSwitch(drvObstr)
	go drivers.PollStopButton(drvStop)

	for {
		select {
		case be := <-drvButtons:
			// Update the request matrix with the new order.
			switch be.Button {
			case drivers.BT_Cab:
				_ = rm.SetCabRequest(be.Floor, true)
			case drivers.BT_HallUp:
				_ = rm.SetHallRequest(be.Floor, 0, true)
			case drivers.BT_HallDown:
				_ = rm.SetHallRequest(be.Floor, 1, true)
			}
			drivers.SetButtonLamp(be.Button, be.Floor, true)
			//rm.DebugPrint()

		case <-drvFloors:
			// Notify the FSM that the elevator has arrived at a floor.
			elevatorFSM.UpdateElevatorState(elevator.EventArrivedAtFloor)

		case obstr := <-drvObstr:
			// Notify the FSM of door obstruction events.
			if obstr {
				elevatorFSM.UpdateElevatorState(elevator.EventDoorObstructed)
			} else {
				elevatorFSM.UpdateElevatorState(elevator.EventDoorReleased)
			}

		case <-drvStop:
			// Clear all button lamps.
			for f := 0; f < numFloors; f++ {
				for b := drivers.ButtonType(0); b < 3; b++ {
					drivers.SetButtonLamp(b, f, false)
				}
			}
		}
	}
}
