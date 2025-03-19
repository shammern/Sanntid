// main.go
package main

import (
	"elevator-project/app"
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/elevator"
	"elevator-project/pkg/message"
	"elevator-project/pkg/network/bcast"
	"flag"
	"fmt"
)

func main() {
	flag.IntVar(&config.ElevatorID, "id", 0, "ElevatorID")
	flag.Parse()

	var msgIDcounter message.MsgID

	drivers.Init(config.ElevatorAddresses[config.ElevatorID], config.NumFloors)

	msgTx := make(chan message.Message)
	msgRx := make(chan message.Message)
	ackChan := make(chan message.Message)
	go bcast.Transmitter(config.BCport, msgTx)
	go bcast.Receiver(config.BCport, msgRx)

	elevator := elevator.NewElevator(config.ElevatorID, msgTx, &msgIDcounter)
	go app.MessageHandler(msgRx, ackChan, msgTx, elevator)
	go app.StartHeartbeatBC(msgTx)
	go elevator.Run()
	go app.MonitorSystemInputs(elevator, msgTx)
	go app.P2Pmonitor()
	go app.StartWorldviewBC(elevator, msgTx, &msgIDcounter)

	if config.ElevatorID == 1 {
		app.IsMaster = true
		fmt.Println("Elevator initated as master")
	}

	select {}
}
