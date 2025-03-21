package app

import (
	"elevator-project/pkg/HRA"
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/elevator"
	"elevator-project/pkg/message"
	"elevator-project/pkg/state"
	"elevator-project/pkg/utils"

	"fmt"
	"time"
)

var IsMaster bool = false
var CurrentMasterID int = 1
var MasterStateStore = state.NewStore()
var msgID = &message.MsgID{}

func MessageHandler(msgRx chan message.Message, ackChan chan message.Message, msgTx chan message.Message, elevatorFSM *elevator.Elevator, trackerChan chan *message.AckTracker) {
	for msg := range msgRx {
		if msg.ElevatorID != config.ElevatorID {

			switch msg.Type {
			case message.Ack:
				ackChan <- msg

			case message.OrderDelegation:
				orderData := msg.OrderData

				fmt.Println("[MH] New orders received, sending to elevator")

				events := convertOrderDataToOrders(orderData)
				for _, event := range events {
					elevatorFSM.Orders <- event
				}

				MasterStateStore.SetAllHallRequest(msg.HallRequests)
				elevatorFSM.SetHallLigths(MasterStateStore.HallRequests)

				SendAck(msg, msgTx)

			case message.CompletedOrder:
				//TODO: Notify
				//fmt.Printf("[MH] Order has been completed: ElevatorID: %d, Floor: %d, ButtonType: %s\n", msg.ElevatorID, msg.ButtonEvent.Floor, utils.ButtonTypeToString(msg.ButtonEvent.Button))
				MasterStateStore.ClearOrder(msg.ButtonEvent, msg.ElevatorID)
				MasterStateStore.ClearHallRequest(msg.ButtonEvent)
				elevatorFSM.SetHallLigths(MasterStateStore.HallRequests)
				SendAck(msg, msgTx)

			case message.ButtonEvent:

				if IsMaster {
					switch msg.ButtonEvent.Button {
					case drivers.BT_HallDown, drivers.BT_HallUp:
						if !MasterStateStore.GetHallOrders()[msg.ButtonEvent.Floor][int(msg.ButtonEvent.Button)] {
							MasterStateStore.SetHallRequest(msg.ButtonEvent)
						}

					case drivers.BT_Cab:
						if !MasterStateStore.Elevators[msg.ElevatorID].RequestMatrix.CabRequests[msg.ButtonEvent.Floor] {
							MasterStateStore.Elevators[msg.ElevatorID].RequestMatrix.CabRequests[msg.ButtonEvent.Floor] = true
						}
					}
					SendAck(msg, msgTx)
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
		MasterStateStore.UpdateStatus(status)
		stateMsg := message.Message{
			Type:       message.State,
			ElevatorID: status.ElevatorID,
			//MsgID:        counter.Next(),
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

			if IsMaster {
				MasterStateStore.SetHallRequest(be)
			} else {
				elevatorFSM.NotifyMaster(message.ButtonEvent, be)
			}

			//If internal event(cab button) add order directly to request matrix
			if be.Button == drivers.BT_Cab {
				order := elevator.Order{
					Event: be,
					Flag:  true,
				}

				elevatorFSM.Orders <- order
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

func SendAck(msg message.Message, msgTx chan message.Message) {
	fmt.Printf("[MH] Message received from elevatorID: %d:\n\tType: %s, msgID: %d, sending ack\n", msg.ElevatorID, utils.MessageTypeToString(msg.Type), msg.MsgID)

	ackMsg := message.Message{
		Type:       message.Ack,
		ElevatorID: config.ElevatorID,
		//MsgID:      msgID.Get(),
		AckID: msg.MsgID,
	}

	msgTx <- ackMsg
}

func SendOrder(newOrder map[string][][2]bool, msgTx chan message.Message, trackChan chan *message.AckTracker) {
	currentMsgID := msgID.Get()
	tracker := message.NewAckTracker(currentMsgID, utils.GetActiveElevators())

	//Acknowlegde own message
	tracker.ExpectedAcks[config.ElevatorID] = true

	trackChan <- tracker

	ticker := time.NewTicker(config.ResendInterval)
	defer ticker.Stop()

	// Sending loop
	orderMsg := message.Message{
		Type:         message.OrderDelegation,
		ElevatorID:   config.ElevatorID,
		MsgID:        msgID.Next(),
		OrderData:    newOrder,
		HallRequests: MasterStateStore.HallRequests,
	}

	for {

		select {
		case <-tracker.Done:
			fmt.Printf("[MH: Master] All acks received for MsgID: %d, stopping order broadcast\n", tracker.MsgID)
			//delete(outstandingAcks, orderMsg.MsgID)
			//TODO: Should delete tracker

			return
		case <-ticker.C:
			fmt.Println("[MH: Master] Resending order for MsgID:", orderMsg.MsgID)
			msgTx <- orderMsg
		}
	}
}

func HRALoop(elevatorFSM *elevator.Elevator, msgTx chan message.Message, trackerChan chan *message.AckTracker) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		if IsMaster {
			newOrder, input, _ := HRA.HRARun(MasterStateStore)
			if !utils.CompareMaps(newOrder, MasterStateStore.CurrentOrders) {
				HRA.PrintHRAInput(input)
				fmt.Println("[HRA] New orders found, sending orders")

				fmt.Println("Master sending the output:")
				for k, v := range newOrder {
					fmt.Printf("%6v : %+v\n", k, v)
				}

				// Launch the SendOrder routine (which itself uses an AckTracker)

				MasterStateStore.CurrentOrders = newOrder
				go SendOrder(newOrder, msgTx, trackerChan)

				// Process the new order events.
				events := convertOrderDataToOrders(newOrder)
				for _, event := range events {
					elevatorFSM.Orders <- event
				}
			}
		}
	}
}
