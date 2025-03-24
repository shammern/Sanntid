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
	"fmt"
	"time"
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
	go bcast.Transmitter(config.BCport, msgTx)
	go bcast.Receiver(config.BCport, msgRx)

	ackMonitor := message.NewAckMonitor(ackTrackerChan, ackChan)
	elevator := elevator.NewElevator(config.ElevatorID, msgTx, msgCounter, ackTrackerChan)
	go elevator.Run()

	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			if config.IsMaster {
				continue // Master shouldn't wait for its own state
			}
			elev, ok := state.MasterStateStore.GetElevator(config.ElevatorID)
			if ok {
				if hasCabCalls(elev.RequestMatrix.CabRequests) {
					fmt.Printf("[DEBUG] Master has stored cab calls for me (Elevator %d): %v\n", config.ElevatorID, elev.RequestMatrix.CabRequests)
					elevator.RestoreCabCalls(elev.RequestMatrix.CabRequests)
					break
				} else {
					fmt.Printf("[DEBUG] Elevator %d state found but no cab calls yet...\n", config.ElevatorID)
				}
			}
		}
	}()
	go ackMonitor.RunAckMonitor()

	go app.MessageHandler(msgRx, ackChan, msgTx, elevator, ackTrackerChan)
	go app.MonitorSystemInputs(elevator)
	go peers.P2Pmonitor(state.MasterStateStore)
	go app.StartWorldviewBC(elevator, msgTx, msgCounter)
	go HRA.HRALoop(elevator, msgTx, ackTrackerChan, msgCounter, orderChan)
	go app.OrderSenderWorker(orderChan, msgTx, ackTrackerChan, msgCounter)

	go app.MonitorMasterHeartbeat(state.MasterStateStore, msgTx)

	select {}
}

func hasCabCalls(cabs []bool) bool {
	for _, v := range cabs {
		if v {
			return true
		}
	}
	return false
}
