package app

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/elevator"
	"elevator-project/pkg/message"
	"elevator-project/pkg/orders"
	"elevator-project/pkg/state"
	"elevator-project/pkg/transport"
	"fmt"
	"net"
	"time"
)

//Currently implemted as a var, migth be changed later to a more suitable format
var masterStateStore = state.NewStore()

func HandleMessage(msg message.Message, addr *net.UDPAddr) {
	switch msg.Type {
	case message.State:
		var status state.ElevatorStatus

		status = state.ElevatorStatus{
			ElevatorID:    msg.ElevatorID,
			State:         msg.StateData.State,
			Direction:     msg.StateData.Direction,
			CurrentFloor:  msg.StateData.CurrentFloor,
			TargetFloor:   msg.StateData.TargetFloor,
			RequestMatrix: msg.StateData.RequestMatrix,
			LastUpdated:   msg.StateData.LastUpdated,
		}

		masterStateStore.UpdateStatus(status)

	case message.Heartbeat:
		masterStateStore.UpdateHeartbeat(msg.ElevatorID)

	default:
		
	}
}

func StartHeartbeat(peerAddrs []string, elevatorID int) {
	ticker := time.NewTicker(100 * time.Millisecond)
	seq := 1
	for range ticker.C {
		hbMsg := message.Message{
			Type:       message.Heartbeat,
			ElevatorID: elevatorID,
			Seq:        seq,
		}

		for _, addr := range peerAddrs {
			if err := transport.SendMessage(hbMsg, addr); err != nil {
				fmt.Printf("Error sending state message (seq %d) to %s: %v\n", seq, addr, err)
			}
		}
		seq++
	}
}

func StartStateSender(e *elevator.Elevator, peerAddrs []string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	seq := 1
	for range ticker.C {

		status := e.GetStatus()
		masterStateStore.UpdateStatus(status)

		stateMsg := message.Message{
			Type:       message.State,
			ElevatorID: status.ElevatorID,
			Seq:        seq,
			StateData: &message.ElevatorState{
				ElevatorID:    status.ElevatorID,
				State:         status.State,
				CurrentFloor:  status.CurrentFloor,
				TargetFloor:   status.TargetFloor,
				LastUpdated:   status.LastUpdated,
				RequestMatrix: status.RequestMatrix,
			},
		}

		for _, addr := range peerAddrs {
			if err := transport.SendMessage(stateMsg, addr); err != nil {
				fmt.Printf("Error sending state message (seq %d) to %s: %v\n", seq, addr, err)
			}
		}
		seq++
	}
}

func DebugPrintStateStore() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Println("----- Current Elevator States -----")
		statuses := masterStateStore.GetAll()
		for id, status := range statuses {
			fmt.Printf("Elevator %d:\n", id)
			fmt.Printf("  ElevatorID   : %d\n", status.ElevatorID)
			fmt.Printf("  State        : %d\n", status.State)
			fmt.Printf("  Direction    : %d\n", status.Direction)
			fmt.Printf("  CurrentFloor : %d\n", status.CurrentFloor)
			fmt.Printf("  TargetFloor  : %d\n", status.TargetFloor)
			fmt.Printf("  LastUpdated  : %v\n", status.LastUpdated.Format("15:04:05"))
			fmt.Printf("  Hallrequests: %+v\n", status.RequestMatrix.HallRequests)
			fmt.Printf("  Cabrequests: %+v\n", status.RequestMatrix.CabRequests)
			fmt.Println()
		}
		fmt.Println("-----------------------------------")
	}
}

func RunEventLoop(elevatorFSM *elevator.Elevator, reqMatrix *orders.RequestMatrix) {
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
			switch be.Button {
			case drivers.BT_Cab:
				_ = reqMatrix.SetCabRequest(be.Floor, true)
			case drivers.BT_HallUp:
				_ = reqMatrix.SetHallRequest(be.Floor, 0, true)
			case drivers.BT_HallDown:
				_ = reqMatrix.SetHallRequest(be.Floor, 1, true)
			}
			drivers.SetButtonLamp(be.Button, be.Floor, true)

		case <-drvFloors:
			elevatorFSM.UpdateElevatorState(elevator.EventArrivedAtFloor)

		case obstr := <-drvObstr:
			if obstr {
				elevatorFSM.UpdateElevatorState(elevator.EventDoorObstructed)
			} else {
				elevatorFSM.UpdateElevatorState(elevator.EventDoorReleased)
			}

		case <-drvStop:
			for f := 0; f < config.NumFloors; f++ {
				for b := drivers.ButtonType(0); b < 3; b++ {
					drivers.SetButtonLamp(b, f, false)
				}
			}
		}
	}
}
