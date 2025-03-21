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
	app.MasterStateStore.UpdateHeartbeat(config.ElevatorID)

	/*go func() { //Kun til debugging, kan fjernes, viser hvem som er masteren
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			fmt.Printf("[Elevator %d] Nåværende master: %d\n", config.ElevatorID, app.CurrentMasterID)
		}
	}()*/

	go app.StartHeartbeatBC(msgTx)
	go elevator.Run()
	go app.MonitorSystemInputs(elevator, msgTx)
	go app.P2Pmonitor(msgTx)
	go app.MonitorMasterHeartbeat(app.MasterStateStore, msgTx)
	select {}
}
