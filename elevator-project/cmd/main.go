package main

import (
	"elevator-project/app"
	"elevator-project/pkg/HRA"
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/elevator"
	"elevator-project/pkg/master"
	"elevator-project/pkg/message"
	"elevator-project/pkg/network/bcast"
	"elevator-project/pkg/network/peers"
	"elevator-project/pkg/systemdata"
	"flag"
)

func main() {
	flag.IntVar(&config.ElevatorID, "id", 0, "ElevatorID")
	flag.Parse()

	drivers.Init(config.ElevatorAddresses[config.ElevatorID], config.NumFloors)
	message.InitMsgCounter()

	msgTx := make(chan message.Message)
	msgRx := make(chan message.Message)
	ackChan := make(chan message.Message)
	ackTrackerChan := make(chan *message.AckTracker)
	orderChan := make(chan HRA.Output)
	masterAnnounced := make(chan struct{}, 1)

	go bcast.Transmitter(config.BCport, msgTx)
	go bcast.Receiver(config.BCport, msgRx)

	ackMonitor := message.NewAckMonitor(ackTrackerChan, ackChan)
	elevator := elevator.NewElevator(config.ElevatorID, msgTx, &message.MsgCounter, ackTrackerChan)

	go ackMonitor.RunAckMonitor()
	go app.MessageHandler(msgRx, ackChan, msgTx, elevator, ackTrackerChan, masterAnnounced)
	go peers.P2Pmonitor(systemdata.MasterStateStore)
	go master.InitMasterDiscovery(msgTx, ackTrackerChan, masterAnnounced)
	go master.MonitorMasterHeartbeat(systemdata.MasterStateStore, msgTx, ackTrackerChan)
	go HRA.HRAWorker(elevator, msgTx, ackTrackerChan, &message.MsgCounter, orderChan)
	go app.OrderSenderWorker(orderChan, msgTx, ackTrackerChan, &message.MsgCounter)
	go app.MonitorSystemInputs(elevator)

	elevator.InitElevator()
	elevator.SetHallLigths(systemdata.MasterStateStore.HallRequests)

	go elevator.Run()
	go app.StartWorldviewBC(elevator, msgTx, &message.MsgCounter)

	select {}
}
