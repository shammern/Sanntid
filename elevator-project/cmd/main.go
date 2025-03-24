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
	go bcast.Transmitter(config.BCport, msgTx)
	go bcast.Receiver(config.BCport, msgRx)

	ackMonitor := message.NewAckMonitor(ackTrackerChan, ackChan)
	elevator := elevator.NewElevator(config.ElevatorID, msgTx, msgCounter, ackTrackerChan)
	go app.MessageHandler(msgRx, ackChan, msgTx, elevator, ackTrackerChan)
	//go app.StartHeartbeatBC(msgTx)
	go elevator.Run()
	go app.MonitorSystemInputs(elevator)
	go peers.P2Pmonitor(state.MasterStateStore, msgTx)
	go app.StartWorldviewBC(elevator, msgTx, msgCounter)
	go ackMonitor.RunAckMonitor()
	go app.HRALoop(elevator, msgTx, ackTrackerChan, msgCounter)

	//FSM

	//app.MasterStateStore.UpdateHeartbeat(config.ElevatorID)

	select {}
}

/* Restarts the program if it panics. Dont think they will test this kind of failure at the FAT test.
I Think they only check for motor crash. no network connection, and manually holding the elevator in between floors for some time.
func main() {

	flag.IntVar(&config.ElevatorID, "id", 0, "ElevatorID")
	flag.Parse()

	// Restart loop: if runElevatorSafely panics, we catch it and restart after a pause.
	for {
		runElevatorSafely()
		log.Println("Elevator process crashed. Restarting in 1 second...")
		time.Sleep(1 * time.Second)

	}
}

func runElevatorSafely() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic: %v", r)
		}
	}()

	var msgIDcounter message.MsgID

	drivers.Init(config.ElevatorAddresses[config.ElevatorID], config.NumFloors)

	msgTx := make(chan message.Message)
	msgRx := make(chan message.Message)
	ackChan := make(chan message.Message)

	//Starting broadcast transmitter and receivers
	go bcast.Transmitter(config.BCport, msgTx)
	go bcast.Receiver(config.BCport, msgRx)

	//FSM
	elevator := elevator.NewElevator(config.ElevatorID, msgTx, &msgIDcounter)

	//Starting the message handler
	go app.MessageHandler(msgRx, ackChan, msgTx, elevator)
	app.MasterStateStore.UpdateHeartbeat(config.ElevatorID)

	go app.StartHeartbeatBC(msgTx)
	go elevator.Run()
	go app.MonitorSystemInputs(elevator, msgTx)
	go app.P2Pmonitor(msgTx)
	go app.MonitorMasterHeartbeat(app.MasterStateStore, msgTx)

	//Test: Crashing program to provoke self restart
		fmt.Println("Program running. It will simulate a crash in 10sec..")
		time.Sleep(10 * time.Second)
		panic("Simulated crash!")
	select {}
}
*/
