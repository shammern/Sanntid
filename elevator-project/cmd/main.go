// main.go
package main

import (
	"elevator-project/app"
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/elevator"
	"elevator-project/pkg/message"
	"elevator-project/pkg/network/bcast"
	"elevator-project/pkg/network/peers"
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
	ackTrackerChan := make(chan *message.AckTracker)
	go bcast.Transmitter(config.BCport, msgTx)
	go bcast.Receiver(config.BCport, msgRx)

	ackMonitor := message.NewAckMonitor(ackTrackerChan, ackChan)
	elevator := elevator.NewElevator(config.ElevatorID, msgTx, &msgIDcounter, ackTrackerChan)
	go app.MessageHandler(msgRx, ackChan, msgTx, elevator, ackTrackerChan)
	//go app.StartHeartbeatBC(msgTx)
	go elevator.Run()
	go app.MonitorSystemInputs(elevator)
	go peers.P2Pmonitor()
	go app.StartWorldviewBC(elevator, msgTx, &msgIDcounter)
	go ackMonitor.RunAckMonitor()
	go app.HRALoop(elevator, msgTx, ackTrackerChan)

	//TODO: implement this in a better way, where it ask the network whos the main, and alternativly promotes itself.
	if config.ElevatorID == 1 {
		app.IsMaster = true
		fmt.Println("Elevator initated as master")
	}

	select {}
}
