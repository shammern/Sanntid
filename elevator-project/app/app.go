package app

import (
	"elevator-project/pkg/HRA"
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/elevator"
	"elevator-project/pkg/message"
	"elevator-project/pkg/network/peers"
	"elevator-project/pkg/state"
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
		fmt.Println("Message received")
		//if msg.ElevatorID != config.ElevatorID {
		switch msg.Type {
		case message.Ack:
			//TODO: check if ack is on correct msg
			if msg.AckID == msgID.Get() {
				fmt.Printf("Received ACK: %#v\n", msg)
				ackChan <- msg
			}

		case message.OrderDelegation:
			orderData := msg.OrderData

			myOrderData := orderData[strconv.Itoa(config.ElevatorID)]
			fmt.Println("My new hallorder are: ")
			for floor, arr := range myOrderData {
				fmt.Printf("  Floor %d: Up: %t, Down: %t\n", floor, arr[0], arr[1])
			}

			events := convertOrderDataToButtonEvents(orderData)
			for _, event := range events {
				elevatorFSM.Orders <- event
			}

			masterStateStore.SetAllHallRequest(msg.HallRequests)
			elevatorFSM.SetHallLigths(masterStateStore.HallRequests)

			//TODO: Handle new order, add to internal request matrix and send ACK back to master
			//fmt.Printf("Received Order: %#v, sending ACK...\n", msg)
			ackMsg := message.Message{
				Type:       message.Ack,
				ElevatorID: config.ElevatorID,
				MsgID:      msgID.Get(),
				AckID:      msg.MsgID,
			}

			msgTx <- ackMsg

		case message.CompletedOrder:
			//TODO: Notify
			fmt.Printf("Order has been completed: Floor: %d, ButtonType: %d\n", msg.ButtonEvent.Floor, int(msg.ButtonEvent.Button))
			masterStateStore.ClearOrder(msg.ButtonEvent, msg.ElevatorID)
			masterStateStore.ClearHallRequest(msg.ButtonEvent)
			masterStateStore.SetAllHallRequest(msg.HallRequests)
			elevatorFSM.SetHallLigths(masterStateStore.HallRequests)

		case message.ButtonEvent:
			fmt.Println("New button event has been received")
			if IsMaster {
				
				//SetHallRequest should maybe be moved to a later stage after an ack has been received.
				//masterStateStore.SetHallRequest(msg.ButtonEvent)
				if msg.ButtonEvent.Button != drivers.BT_Cab {
					fmt.Println("Master has received new order")
					masterStateStore.SetHallRequest(msg.ButtonEvent)
					newOrder, _ := HRA.HRARun(masterStateStore)
					orderMsg := message.Message{
						Type:       message.OrderDelegation,
						ElevatorID: config.ElevatorID,
						MsgID:      msgID.Next(),
						AckID:      msg.MsgID,
						OrderData:  newOrder,
					}

					msgTx <- orderMsg
				}

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
			//masterStateStore.HallRequests = msg.HallRequests
			//elevatorFSM.SetHallLigths(masterStateStore.HallRequests)

		case message.MasterSlaveConfig:
			// Update our view of the current master.
			fmt.Printf("Received master config update: new master is elevator %d\n", msg.ElevatorID)
			CurrentMasterID = msg.ElevatorID
			if config.ElevatorID != msg.ElevatorID {
				IsMaster = false
			} else {
				IsMaster = true
			}

		default:
			fmt.Printf("Received message: %#v\n", msg)
		}
	}
	//}
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

func MonitorSystemInputs(elevatorFSM *elevator.Elevator, msgTx chan message.Message) {
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
			//BC buttonevent on network
			buttonEventMsg := message.Message{
				Type:        message.ButtonEvent,
				ElevatorID:  config.ElevatorID,
				MsgID:       msgID.Next(),
				ButtonEvent: be,
			}

			fmt.Println("Sending order on network")
			msgTx <- buttonEventMsg

			//If internal event(cab button) add order directly to request matrix
			if be.Button == drivers.BT_Cab {
				elevatorFSM.Orders <- be
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

// TODO: Fix this function
func StartMasterProcess(peerAddrs []string, elevatorFSM *elevator.Elevator, msgTx chan message.Message) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	/*
		for range ticker.C {
			if elevatorFSM.ElevatorID == 1 { // Master elevator check
				fmt.Println("[Master] Checking for unassigned orders...")

				unassignedOrders := orders.GetUnassignedOrders(elevatorFSM.GetRequestMatrix())
				fmt.Println("[Master] Unassigned Orders:", unassignedOrders)

				for _, order := range unassignedOrders {
					assignedElevator := orders.FindBestElevator(order, peerAddrs)
					fmt.Printf("[Master] Assigning order: Floor %d to Elevator %s\n", order.Floor, assignedElevator)


					if assignedElevator != "" {
						orderMsg := message.Message{
							Type:       message.OrderDelegation,
							ElevatorID: status.ElevatorID,
							MsgID:      msgID,
							OrderData: {},
							},
						msgTx <-
						transport.SendOrderToElevator(order, assignedElevator)
					}
				}
			}
		}

	*/
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
