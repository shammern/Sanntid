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

	ch_msgTx := make(chan message.Message)
	ch_msgRx := make(chan message.Message)
	ch_ack := make(chan message.Message)
	ch_ackTracker := make(chan *message.AckTracker)
	ch_order := make(chan HRA.Output)
	ch_masterAnnounced := make(chan struct{}, 1)

	go bcast.Transmitter(config.BCport, ch_msgTx)
	go bcast.Receiver(config.BCport, ch_msgRx)

	ackMonitor := message.NewAckMonitor(ch_ackTracker, ch_ack)
	elevator := elevator.NewElevator(config.ElevatorID, ch_msgTx, &message.MsgCounter, ch_ackTracker)

	go ackMonitor.RunAckMonitor()
	go app.MessageHandler(ch_msgRx, ch_ack, ch_msgTx, elevator, ch_ackTracker, ch_masterAnnounced)
	go peers.P2Pmonitor(systemdata.MasterStateStore)
	go master.InitMasterDiscovery(ch_msgTx, ch_ackTracker, ch_masterAnnounced)
	go master.MonitorMasterHeartbeat(systemdata.MasterStateStore, ch_msgTx, ch_ackTracker)
	go HRA.HRAWorker(elevator, ch_msgTx, ch_ackTracker, &message.MsgCounter, ch_order)
	go app.OrderSenderWorker(ch_order, ch_msgTx, ch_ackTracker, &message.MsgCounter)
	go app.MonitorSystemInputs(elevator)

	elevator.InitElevator()
	elevator.SetHallLigths(systemdata.MasterStateStore.HallRequests)

	go elevator.Run()
	go app.StartWorldviewBC(elevator, ch_msgTx, &message.MsgCounter)

	select {}
}
