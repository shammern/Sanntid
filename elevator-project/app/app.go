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

var CurrentMasterID int = -1
var BackupElevatorID int = -1

func MessageHandler(msgRx chan message.Message, ackChan chan message.Message, msgTx chan message.Message, elevatorFSM *elevator.Elevator, trackerChan chan *message.AckTracker) {
	for msg := range msgRx {
		if msg.ElevatorID != config.ElevatorID {

			switch msg.Type {
			case message.Ack:
				ackChan <- msg

			case message.OrderDelegation:
				orderData := msg.OrderData

				fmt.Println("[MH] New orders received, sending to elevator")

				events := elevator.ConvertOrderDataToOrders(orderData)
				for _, event := range events {
					elevatorFSM.Orders <- event
				}

				state.MasterStateStore.SetAllHallRequest(msg.HallRequests)
				elevatorFSM.SetHallLigths(state.MasterStateStore.HallRequests)

				SendAck(msg, msgTx)

			case message.CompletedOrder:
				//TODO: Notify
				SendAck(msg, msgTx)
				fmt.Printf("[MH] Order has been completed: ElevatorID: %d, Floor: %d, ButtonType: %s\n", msg.ElevatorID, msg.ButtonEvent.Floor, utils.ButtonTypeToString(msg.ButtonEvent.Button))
				state.MasterStateStore.ClearOrder(msg.ButtonEvent, msg.ElevatorID)
				state.MasterStateStore.ClearHallRequest(msg.ButtonEvent)
				elevatorFSM.SetHallLigths(state.MasterStateStore.HallRequests)

			case message.ButtonEvent:
				if config.IsMaster {
					fmt.Printf("[MASTER] Received ButtonEvent: Elevator %d | Floor %d | Type: %v\n",
						msg.ElevatorID, msg.ButtonEvent.Floor, msg.ButtonEvent.Button)

					switch msg.ButtonEvent.Button {
					case drivers.BT_HallUp, drivers.BT_HallDown:
						state.MasterStateStore.SetHallRequest(msg.ButtonEvent)

					case drivers.BT_Cab:
						if elev, ok := state.MasterStateStore.GetElevator(msg.ElevatorID); ok {
							elev.RequestMatrix.CabRequests[msg.ButtonEvent.Floor] = true
							state.MasterStateStore.UpdateStatus(elev) // ✅ Safe write
							fmt.Printf("[MASTER] ✅ Stored cab request: Elevator %d, Floor %d\n",
								msg.ElevatorID, msg.ButtonEvent.Floor)
						} else {
							fmt.Printf("[MASTER] ⚠️ Elevator %d not found in MasterStateStore\n", msg.ElevatorID)
						}

					}
				}

				SendAck(msg, msgTx)

			case message.State:
				status := state.ElevatorStatus{
					ElevatorID:      msg.ElevatorID,
					State:           msg.StateData.State,
					CurrentFloor:    msg.StateData.CurrentFloor,
					TravelDirection: msg.StateData.TravelDirection,
					RequestMatrix:   msg.StateData.RequestMatrix,
					LastUpdated:     msg.StateData.LastUpdated,
					Available:       msg.StateData.Available,
				}
				existingStatus, ok := state.MasterStateStore.GetElevator(msg.ElevatorID)
				if ok {
					for floor, storedCall := range existingStatus.RequestMatrix.CabRequests {
						if storedCall && !status.RequestMatrix.CabRequests[floor] {
							status.RequestMatrix.CabRequests[floor] = true
						}
					}
				}
				fmt.Printf("[MASTER DEBUG] Updated state for Elevator %d, cab calls: %v\n", msg.ElevatorID, status.RequestMatrix.CabRequests)
				state.MasterStateStore.UpdateStatus(status)

			case message.MasterQuery:
				// Update our view of the current master.
				fmt.Printf("[MH] Received master config update: new master is elevator %d\n", msg.ElevatorID)
				CurrentMasterID = msg.ElevatorID
				if config.ElevatorID != msg.ElevatorID {
					config.IsMaster = false
				} else {
					config.IsMaster = true
				}

			case message.MasterAnnouncement:
				fmt.Printf("[INFO] Oppdaterer master til heis %d\n", msg.MasterID)
				CurrentMasterID = msg.MasterID
				config.IsMaster = (config.ElevatorID == CurrentMasterID)
			}
		}
	}
}

func StartWorldviewBC(e *elevator.Elevator, msgTx chan message.Message, counter *message.MsgID) {
	ticker := time.NewTicker(config.WorldviewBCInterval)
	defer ticker.Stop()

	for range ticker.C {
		status := e.GetStatus()
		state.MasterStateStore.UpdateStatus(status)
		stateMsg := message.Message{
			Type:         message.State,
			ElevatorID:   status.ElevatorID,
			HallRequests: state.MasterStateStore.HallRequests,

			StateData: &message.ElevatorState{
				ElevatorID:      status.ElevatorID,
				State:           status.State,
				Available:       status.Available,
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
		statuses := state.MasterStateStore.GetAll()

		for id, status := range statuses {
			fmt.Printf("Elevator %d:\n", id)
			fmt.Printf("  ElevatorID   : %d\n", status.ElevatorID)
			//fmt.Printf("  State        : %s\n", HRA.StateIntToString(status.State))
			fmt.Printf("  Available        : %t\n", status.Available)
			fmt.Printf("  CurrentFloor : %d\n", status.CurrentFloor)
			//fmt.Printf("  TravelingDirection  : %s\n", HRA.DirectionIntToString(status.TravelDirection))
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

			if be.Button == drivers.BT_Cab {
				order := elevator.Order{
					Event: be,
					Flag:  true,
				}
				elevatorFSM.Orders <- order
				drivers.SetButtonLamp(drivers.BT_Cab, be.Floor, true)

				// 👇 This is the fix: tell the master!
				if !config.IsMaster {
					fmt.Printf("[DEBUG] Sending cab call to master: floor %d, elevator %d\n", be.Floor, config.ElevatorID)
					elevatorFSM.NotifyMaster(message.ButtonEvent, be)
				}
			} else {
				// Only hall calls here
				if config.IsMaster {
					state.MasterStateStore.SetHallRequest(be)
				} else {
					elevatorFSM.NotifyMaster(message.ButtonEvent, be)
				}
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
	fmt.Printf("[MH] Message received from elevatorID: %d:\n\tType: %s, msgID: %s, sending ack\n", msg.ElevatorID, utils.MessageTypeToString(msg.Type), msg.MsgID)

	ackMsg := message.Message{
		Type:       message.Ack,
		ElevatorID: config.ElevatorID,
		//MsgID:      msgID.Get(),
		AckID: msg.MsgID,
	}

	msgTx <- ackMsg
}

func SendOrder(newOrder map[string][][2]bool, msgTx chan message.Message, trackChan chan *message.AckTracker, msgID *message.MsgID) {
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
		HallRequests: state.MasterStateStore.HallRequests,
	}

	for {

		select {
		case <-tracker.Done:
			fmt.Printf("[MH: Master] Terminating order broadcast for MsgID: %s\n", tracker.MsgID)
			return

		case <-ticker.C:
			fmt.Println("[MH: Master] Resending order for MsgID:", orderMsg.MsgID)
			msgTx <- orderMsg
		}
	}
}

// orderSenderWorker continuously listens for order requests and handles cancellation/resending.
func OrderSenderWorker(orderRequestCh <-chan HRA.OrderData, msgTx chan message.Message, trackerChan chan *message.AckTracker, msgID *message.MsgID) {
	var currentTracker *message.AckTracker
	ticker := time.NewTicker(config.ResendInterval)
	defer ticker.Stop()

	// Use a for-select loop that also listens for new orders.
	for {
		select {
		// New order received from HRALoop.
		case req := <-orderRequestCh:
			// Cancel previous order if any.
			if currentTracker != nil {
				currentTracker.Terminate()
			}
			// Create a new tracker.
			currentMsgID := msgID.Get()
			tracker := message.NewAckTracker(currentMsgID, utils.GetActiveElevators())
			tracker.ExpectedAcks[config.ElevatorID] = true
			trackerChan <- tracker
			currentTracker = tracker

			// Prepare the message that will be repeatedly sent.
			orderMsg := message.Message{
				Type:         message.OrderDelegation,
				ElevatorID:   config.ElevatorID,
				MsgID:        msgID.Next(),
				OrderData:    req.Orders,
				HallRequests: state.MasterStateStore.HallRequests,
			}

			// Enter a nested loop to resend the current order.
		resendLoop:
			for {
				select {
				case <-tracker.Done:
					fmt.Printf("[MH: Master] Terminating sending of MsgID: %s, stopping order broadcast\n", tracker.MsgID)
					break resendLoop

				case <-ticker.C:
					fmt.Println("[MH: Master] Resending order for MsgID:", orderMsg.MsgID)
					msgTx <- orderMsg

				case newReq := <-orderRequestCh:
					currentTracker.Terminate()

					req = newReq
					break resendLoop
				}
			}
		}
	}
}
