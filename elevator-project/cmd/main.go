// main.go
package main

import (
	"elevator-project/app"
	"elevator-project/pkg/HRA"
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/elevator"
	"elevator-project/pkg/message"
	"elevator-project/pkg/network/bcast"
	"elevator-project/pkg/network/peers"
	"elevator-project/pkg/state"
	"flag"
)

func main() {

	flag.IntVar(&config.ElevatorID, "id", 0, "ElevatorID")
	flag.Parse()

	msgCounter := message.NewMsgId()

	drivers.Init(config.ElevatorAddresses[config.ElevatorID], config.NumFloors)

	msgTx := make(chan message.Message)
	msgRx := make(chan message.Message)
	ackChan := make(chan message.Message)
	ackTrackerChan := make(chan *message.AckTracker)
	orderChan := make(chan HRA.OrderData)
	masterAnnounced := make(chan struct{}, 1) //Buffer size
	go bcast.Transmitter(config.BCport, msgTx)
	go bcast.Receiver(config.BCport, msgRx)

	ackMonitor := message.NewAckMonitor(ackTrackerChan, ackChan)
	elevator := elevator.NewElevator(config.ElevatorID, msgTx, msgCounter, ackTrackerChan)

	go elevator.Run()
	go ackMonitor.RunAckMonitor()
	go app.MessageHandler(msgRx, ackChan, msgTx, elevator, ackTrackerChan, masterAnnounced)
	go app.MonitorSystemInputs(elevator)
	go peers.P2Pmonitor(state.MasterStateStore)
	go app.StartWorldviewBC(elevator, msgTx, msgCounter)
	go app.InitMasterDiscovery(msgTx, masterAnnounced)
	go HRA.HRALoop(elevator, msgTx, ackTrackerChan, msgCounter, orderChan)
	go app.OrderSenderWorker(orderChan, msgTx, ackTrackerChan, msgCounter)

	go app.MonitorMasterHeartbeat(state.MasterStateStore, msgTx)

	select {}
}
