package app

import (
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
var CurrentMasterID int = -1
var BackupElevatorID int = -1
var MasterStateStore = state.NewStore()
var msgID int = 0          //HMMMM
var Peers peers.PeerUpdate //to maintain elevators in network

func MessageHandler(msgRx chan message.Message, ackChan chan message.Message, msgTx chan message.Message, elevatorFSM *elevator.Elevator) {
	for msg := range msgRx {
		if msg.Type == message.Heartbeat {
			MasterStateStore.UpdateHeartbeat(msg.ElevatorID)
			continue
		}
		if msg.ElevatorID != config.ElevatorID {
			switch msg.Type {
			case message.Ack:
				//TODO: check if ack is on correct msg
				if msg.AckID == msgID {
					fmt.Printf("Received ACK: %#v\n", msg)
					ackChan <- msg
				}

			case message.OrderDelegation:
				//TODO: Handle new order, add to internal request matrix and send ACK back to master
				//fmt.Printf("Received Order: %#v, sending ACK...\n", msg)
				ackMsg := message.Message{
					Type:       message.Ack,
					ElevatorID: config.ElevatorID,
					MsgID:      msgID,
					AckID:      msg.MsgID,
				}
				msgTx <- ackMsg
				//elevatorFSM.RequestMatrix.SetHallRequest(msg.OrderData.Floor, int(msg.OrderData.Button), true)

			case message.MasterQuery: //If the elevator that is running this code is the master, give an announcement to the other elevators
				if IsMaster {
					fmt.Printf("[INFO] Mottok MasterQuery. Jeg er master (heis %d), svarer...\n", CurrentMasterID)
					responseMsg := message.Message{
						Type:       message.MasterAnnouncement,
						ElevatorID: CurrentMasterID,
					}
					msgTx <- responseMsg
				}
				continue

			case message.CompletedOrder:
				//TODO: Notify
				MasterStateStore.ClearHallLight(msg.OrderData.Floor, int(msg.OrderData.Button))
				elevatorFSM.SetHallLigths(MasterStateStore.GetElevatorLights(config.ElevatorID))

			case message.ButtonEvent:
				//TODO: This has to be linked to the masterroutine that deligates orders
				//Currentstate: updates statestore.Lights and light the appropriate hall lights. Cab buttons are handled internally
				MasterStateStore.SetHallLight(msg.OrderData.Floor, int(msg.OrderData.Button))
				elevatorFSM.SetHallLigths(MasterStateStore.GetElevatorLights(config.ElevatorID))

			case message.State:
				status := state.ElevatorStatus{
					ElevatorID:    msg.ElevatorID,
					State:         msg.StateData.State,
					Direction:     msg.StateData.Direction,
					CurrentFloor:  msg.StateData.CurrentFloor,
					TargetFloor:   msg.StateData.TargetFloor,
					RequestMatrix: msg.StateData.RequestMatrix,
					LastUpdated:   msg.StateData.LastUpdated,
				}
				MasterStateStore.UpdateStatus(status)

			case message.MasterAnnouncement:
				fmt.Printf("[INFO] Oppdaterer master til heis %d\n", msg.ElevatorID)
				CurrentMasterID = msg.ElevatorID
				IsMaster = (config.ElevatorID == msg.ElevatorID)
			default:
				fmt.Printf("Received message: %#v\n", msg)
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
			MsgID:      msgID,
		}
		msgTx <- hbMsg
		msgID++
	}
}

func StartWorldviewBC(e *elevator.Elevator, msgRx chan message.Message, counter *message.MsgID) {
	ticker := time.NewTicker(config.WorldviewBCInterval)
	defer ticker.Stop()

	for range ticker.C {
		status := e.GetStatus()
		MasterStateStore.UpdateStatus(status)
		stateMsg := message.Message{
			Type:       message.State,
			ElevatorID: status.ElevatorID,
			MsgID:      counter.Next(),
			StateData: &message.ElevatorState{
				ElevatorID:    status.ElevatorID,
				State:         status.State,
				CurrentFloor:  status.CurrentFloor,
				TargetFloor:   status.TargetFloor,
				LastUpdated:   time.Now(),
				RequestMatrix: status.RequestMatrix,
			},
		}

		msgRx <- stateMsg
	}
}

func DebugPrintStateStore() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		fmt.Println("----- Current Elevator States -----")
		statuses := MasterStateStore.GetAll()
		for id, status := range statuses {
			fmt.Printf("Elevator %d:\n", id)
			fmt.Printf("  ElevatorID   : %d\n", status.ElevatorID)
			fmt.Printf("  State        : %d\n", status.State)
			fmt.Printf("  Direction    : %d\n", status.Direction)
			fmt.Printf("  CurrentFloor : %d\n", status.CurrentFloor)
			fmt.Printf("  TargetFloor  : %d\n", status.TargetFloor)
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
				Type:       message.ButtonEvent,
				ElevatorID: config.ElevatorID,
				MsgID:      msgID,
				OrderData:  be,
			}

			fmt.Println("Button event triggered, sending message")
			msgTx <- buttonEventMsg

			//If internal event(cab button) add order directly to request matrix
			if be.Button == drivers.BT_Cab {
				_ = elevatorFSM.RequestMatrix.SetCabRequest(be.Floor, true)
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

// This function can be used to trigger events if units exit or enter the network
func P2Pmonitor(msgTx chan message.Message) {
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
		if len(update.New) > 0 {
			fmt.Printf("[INFO] Ny heis oppdaget: %q. Spør etter master...\n", update.New)

			queryMsg := message.Message{
				Type:       message.MasterQuery,
				ElevatorID: 0, //Asking for the master
			}
			msgTx <- queryMsg
		}
	}
}
