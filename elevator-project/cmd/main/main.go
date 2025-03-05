package main

import (
	"elevator-project/app"
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/elevator"
	"elevator-project/pkg/orders"
	"elevator-project/pkg/transport"
	"elevator-project/pkg/utils"
	"flag"
)

func main() {
	//Collecting flags and parsing them to handle correct setup of multiple elevators running on the same computer
	pelevatorID := flag.Int("ID", 0, "elevatorID")
	flag.Parse()

	drivers.Init(config.ElevatorAddresses[*pelevatorID], config.NumFloors)

	peerAddrs := utils.GetOtherElevatorAddresses(*pelevatorID)
	requestMatrix := orders.NewRequestMatrix(config.NumFloors)
	elevatorFSM := elevator.NewElevator(requestMatrix, *pelevatorID)

	go elevatorFSM.Run()
	go transport.StartServer(config.UDPAddresses[*pelevatorID], app.HandleMessage)
	go app.StartHeartbeat(peerAddrs, *pelevatorID)
	go app.StartStateSender(elevatorFSM, peerAddrs)

	app.RunEventLoop(elevatorFSM, requestMatrix)
}
