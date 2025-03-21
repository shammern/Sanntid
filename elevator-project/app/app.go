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
var CurrentMasterID int = -1
var BackupElevatorID int = -1
var MasterStateStore = state.NewStore()
var msgID = &message.MsgID{}
var Peers peers.PeerUpdate //to maintain elevators in network

func MessageHandler(msgRx chan message.Message, ackChan chan message.Message, msgTx chan message.Message, elevatorFSM *elevator.Elevator) {
	for msg := range msgRx {
		//if msg.ElevatorID != config.ElevatorID {
		switch msg.Type {
		case message.Ack:
			//TODO: check if ack is on correct msg
			if msg.AckID == msgID.Get() {
				fmt.Printf("[MH] Received ACK: %#v\n", msg)
				ackChan <- msg
			}
			if msg.Type == message.Heartbeat {
				MasterStateStore.UpdateHeartbeat(msg.ElevatorID)
				continue
			}
			if msg.ElevatorID != config.ElevatorID {
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
					fmt.Println("[MH] My new hallorders are: ")
					for floor, arr := range myOrderData {
						fmt.Printf("  Floor %d: Up: %t, Down: %t\n", floor, arr[0], arr[1])
					}

					events := convertOrderDataToOrders(orderData)
					for _, event := range events {
						//fmt.Printf("[MH] Sending order to FSM: floor: %d, button: %d\n", event.Floor, int(event.Button))
						elevatorFSM.Orders <- event
					}

					MasterStateStore.SetAllHallRequest(msg.HallRequests)
					elevatorFSM.SetHallLigths(MasterStateStore.HallRequests)

					//TODO: Handle new order, add to internal request matrix and send ACK back to master

					ackMsg := message.Message{
						Type:       message.Ack,
						ElevatorID: config.ElevatorID,
						MsgID:      msgID.Get(),
						AckID:      msg.MsgID,
					}

					msgTx <- ackMsg

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
					fmt.Printf("[MH] Order has been completed: Floor: %d, ButtonType: %d\n", msg.ButtonEvent.Floor, int(msg.ButtonEvent.Button))
					MasterStateStore.ClearOrder(msg.ButtonEvent, msg.ElevatorID)
					MasterStateStore.ClearHallRequest(msg.ButtonEvent)
					elevatorFSM.SetHallLigths(MasterStateStore.HallRequests)

				case message.ButtonEvent:

					if IsMaster {

						//SetHallRequest should maybe be moved to a later stage after an ack has been received.
						//MasterStateStore.SetHallRequest(msg.ButtonEvent)
						if msg.ButtonEvent.Button != drivers.BT_Cab {
							MasterStateStore.SetHallRequest(msg.ButtonEvent)
							newOrder, _ := HRA.HRARun(MasterStateStore)
							orderMsg := message.Message{
								Type:         message.OrderDelegation,
								ElevatorID:   config.ElevatorID,
								MsgID:        msgID.Next(),
								AckID:        msg.MsgID,
								OrderData:    newOrder,
								HallRequests: MasterStateStore.HallRequests,
							}

							msgTx <- orderMsg
						}

					}

				case message.Heartbeat:
					MasterStateStore.UpdateHeartbeat(msg.ElevatorID)

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
					MasterStateStore.UpdateStatus(status)
				//MasterStateStore.HallRequests = msg.HallRequests
				//elevatorFSM.SetHallLigths(MasterStateStore.HallRequests)

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
		MasterStateStore.UpdateStatus(status)
		stateMsg := message.Message{
			Type:         message.State,
			ElevatorID:   status.ElevatorID,
			MsgID:        counter.Next(),
			HallRequests: MasterStateStore.HallRequests,
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
		statuses := MasterStateStore.GetAll()
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

			msgTx <- buttonEventMsg

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
