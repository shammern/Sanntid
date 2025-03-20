package app

import (
	"elevator-project/pkg/HRA"
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/elevator"
	"elevator-project/pkg/message"
	"elevator-project/pkg/network/peers"
	"elevator-project/pkg/state"
	"elevator-project/pkg/utils"
	"fmt"
	"strconv"
	"time"
)

var IsMaster bool = false
var CurrentMasterID int = 1
var masterStateStore = state.NewStore()
var msgID = &message.MsgID{}
var Peers peers.PeerUpdate //to maintain elevators in network

func MessageHandler(msgRx chan message.Message, ackChan chan message.Message, msgTx chan message.Message, elevatorFSM *elevator.Elevator) {
	for msg := range msgRx { 
		if msg.ElevatorID != config.ElevatorID {
		
		switch msg.Type {
		case message.Ack:
			fmt.Printf("[MH] Received an ACK\n")
			//TODO: check if ack is on correct msg
			if msg.AckID == msgID.Get() {
				fmt.Printf("[MH] Received correct ACK\n")
				ackChan <- msg
			}

		case message.OrderDelegation:
			orderData := msg.OrderData

			fmt.Println("[MH] New orders received, sending to elevator")
			
			events := convertOrderDataToOrders(orderData)
			for _, event := range events {
				elevatorFSM.Orders <- event
			}

			masterStateStore.SetAllHallRequest(msg.HallRequests)
			elevatorFSM.SetHallLigths(masterStateStore.HallRequests)
			
			SendAck(msg, msgTx)

		case message.CompletedOrder:
			//TODO: Notify
			fmt.Printf("[MH] Order has been completed: Floor: %d, ButtonType: %s\n", msg.ButtonEvent.Floor, utils.ButtonTypeToString(msg.ButtonEvent.Button))
			masterStateStore.ClearOrder(msg.ButtonEvent, msg.ElevatorID)
			masterStateStore.ClearHallRequest(msg.ButtonEvent)
			elevatorFSM.SetHallLigths(masterStateStore.HallRequests)
			SendAck(msg, msgTx)

		case message.ButtonEvent:
			
			if IsMaster {

				//Only want to assign hallorders -> ignore if buttonevent is of type cab.
				if msg.ButtonEvent.Button != drivers.BT_Cab {
					masterStateStore.SetHallRequest(msg.ButtonEvent)
					newOrder, _ := HRA.HRARun(masterStateStore)
					go SendOrder(newOrder, msgTx, ackChan)
					//TODO: Add to own RM

					
					events := convertOrderDataToOrders(newOrder)
					for _, event := range events {
						elevatorFSM.Orders <- event
					}
				}
				
				SendAck(msg, msgTx)

			}

		case message.Heartbeat:
			masterStateStore.UpdateHeartbeat(msg.ElevatorID)

		case message.State:
			status := state.ElevatorStatus{
				ElevatorID:      msg.ElevatorID,
				State:           msg.StateData.State,
				Direction:       msg.StateData.Direction,
				CurrentFloor:    msg.StateData.CurrentFloor,
				TravelDirection: msg.StateData.TravelDirection,
				RequestMatrix:   msg.StateData.RequestMatrix,
				LastUpdated:     msg.StateData.LastUpdated,
			}
			masterStateStore.UpdateStatus(status)

		case message.MasterSlaveConfig:
			// Update our view of the current master.
			fmt.Printf("[MH] Received master config update: new master is elevator %d\n", msg.ElevatorID)
			CurrentMasterID = msg.ElevatorID
			if config.ElevatorID != msg.ElevatorID {
				IsMaster = false
			} else {
				IsMaster = true
			}

		default:
			fmt.Printf("Received undefined message")
		}
	}
	}
}

func StartHeartbeatBC(msgTx chan message.Message) {
	ticker := time.NewTicker(config.HeartBeatInterval)

	for range ticker.C {
		hbMsg := message.Message{
			Type:       message.Heartbeat,
			ElevatorID: config.ElevatorID,
			MsgID:      msgID.Next(),
		}
		msgTx <- hbMsg
	}
}

func StartWorldviewBC(e *elevator.Elevator, msgTx chan message.Message, counter *message.MsgID) {
	ticker := time.NewTicker(config.WorldviewBCInterval)
	defer ticker.Stop()

	for range ticker.C {
		status := e.GetStatus()
		masterStateStore.UpdateStatus(status)
		stateMsg := message.Message{
			Type:       message.State,
			ElevatorID: status.ElevatorID,
			MsgID:      counter.Next(),
			HallRequests: masterStateStore.HallRequests,
			StateData: &message.ElevatorState{
				ElevatorID:      status.ElevatorID,
				State:           status.State,
				CurrentFloor:    status.CurrentFloor,
				TravelDirection: status.TravelDirection,
				LastUpdated:     time.Now(),
				RequestMatrix:   status.RequestMatrix,
			},
		}

		msgTx <- stateMsg
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
			fmt.Printf("  TravelingDirection  : %d\n", status.TravelDirection)
			fmt.Printf("  LastUpdated  : %v\n", status.LastUpdated.Format("15:04:05"))
			fmt.Printf("  HallRequests : %+v\n", status.RequestMatrix.HallRequests)
			fmt.Printf("  CabRequests  : %+v\n", status.RequestMatrix.CabRequests)
			fmt.Println()
		}
		fmt.Println("-----------------------------------")
	}
}

func MonitorSystemInputs(elevatorFSM *elevator.Elevator) {
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
			elevatorFSM.NotifyMaster(message.ButtonEvent, be)

			//If internal event(cab button) add order directly to request matrix
			if be.Button == drivers.BT_Cab {
				elevatorFSM.Orders <- elevator.Order{be, true}
				drivers.SetButtonLamp(drivers.BT_Cab, be.Floor, true)
			}

		case <-drvFloors:
			elevatorFSM.UpdateElevatorState(elevator.EventArrivedAtFloor)

		case obstr := <-drvObstr:
			if obstr {
				elevatorFSM.UpdateElevatorState(elevator.EventDoorObstructed)
			} else {
				elevatorFSM.UpdateElevatorState(elevator.EventDoorReleased)
			}

		case <-drvStop:
			//TODO: Implemnt stop logic
		}
	}
}

func P2Pmonitor() {
	//This function can be used to trigger events if units exit or enter the network
	peerUpdateCh := make(chan peers.PeerUpdate)
	peerTxEnable := make(chan bool)
	go peers.Transmitter(config.P2Pport, strconv.Itoa(config.ElevatorID), peerTxEnable)
	go peers.Receiver(config.P2Pport, peerUpdateCh)
	for {
		update := <-peerUpdateCh
		Peers = update
		fmt.Printf("Peer update:\n")
		fmt.Printf("  Peers:    %q\n", update.Peers)
		fmt.Printf("  New:      %q\n", update.New)
		fmt.Printf("  Lost:     %q\n", update.Lost)
	}
}

func SendAck(msg message.Message, msgTx chan message.Message) {
	fmt.Printf("[MH] Message received: type: %s, msgID: %d, sending ack\n", utils.MessageTypeToString(msg.Type), msg.MsgID)

	ackMsg := message.Message{
		Type:       message.Ack,
		ElevatorID: config.ElevatorID,
		//MsgID:      msgID.Get(),
		AckID:      msg.MsgID,
	}

	msgTx <- ackMsg
}

/*
func SendOrder(newOrder map[string][][2]bool, msgTx chan message.Message, ackChan chan message.Message){
	orderMsg := message.Message{
		Type:       message.OrderDelegation,
		ElevatorID: config.ElevatorID,
		MsgID:      msgID.Next(),
		OrderData:  newOrder,
		HallRequests: masterStateStore.HallRequests,
	}


	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		// If an acknowledgement is received, break out of the loop.
		//TODO: Master need to keep track of which of the online elevators has sent an ack msg and break of of loop when all active elevators has sent an ack. 
		case <- ackChan:
			return
	
		// Otherwise, on each tick, send the message.
		case <-ticker.C:
			fmt.Printf("[MH: Master] Sending new orders.\n")
			msgTx <- orderMsg
		}
	}
}
*/

func SendOrder(newOrder map[string][][2]bool, msgTx chan message.Message, ackChan chan message.Message, expectedPeers []int) {
	orderMsg := message.Message{
		Type:         message.OrderDelegation,
		ElevatorID:   config.ElevatorID,
		MsgID:        msgID.Next(),
		OrderData:    newOrder,
		HallRequests: masterStateStore.HallRequests,
	}

	// Build a map to track ack status for each expected peer.
	ackReceived := make(map[int]bool)
	for _, peerID := range expectedPeers {
		ackReceived[peerID] = false
	}

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		// Check if all expected peers have acknowledged.
		allAcked := true
		for _, ack := range ackReceived {
			if !ack {
				allAcked = false
				break
			}
		}
		if allAcked {
			fmt.Println("[MH: Master] All acknowledgements received.")
			return
		}

		select {
		// Process incoming ack messages.
		case ackMsg := <-ackChan:
			// Use ackMsg.ElevatorID (an int) to identify which peer sent the ack.
			if _, exists := ackReceived[ackMsg.ElevatorID]; exists {
				ackReceived[ackMsg.ElevatorID] = true
				fmt.Printf("[MH: Master] Ack received from elevator: %d\n", ackMsg.ElevatorID)
			} else {
				fmt.Printf("[MH: Master] Received ack from unknown elevator: %d\n", ackMsg.ElevatorID)
			}
		// On each tick, resend the order message.
		case <-ticker.C:
			fmt.Println("[MH: Master] Sending new orders.")
			msgTx <- orderMsg
		}
	}
}

