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
var MasterQueryTimer *time.Timer
var MasterQuerySent bool = false

func MessageHandler(msgRx chan message.Message, ackChan chan message.Message, msgTx chan message.Message, elevatorFSM *elevator.Elevator, trackerChan chan *message.AckTracker) {
	for msg := range msgRx {
		if msg.ElevatorID != config.ElevatorID {

			switch msg.Type {
			case message.Ack:
				ackChan <- msg

			case message.OrderDelegation:
				orderData := msg.OrderData

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
					switch msg.ButtonEvent.Button {
					case drivers.BT_HallDown, drivers.BT_HallUp:
						if !state.MasterStateStore.GetHallOrders()[msg.ButtonEvent.Floor][int(msg.ButtonEvent.Button)] {
							state.MasterStateStore.SetHallRequest(msg.ButtonEvent)
						}

					case drivers.BT_Cab:
						if !state.MasterStateStore.Elevators[msg.ElevatorID].RequestMatrix.CabRequests[msg.ButtonEvent.Floor] {
							state.MasterStateStore.Elevators[msg.ElevatorID].RequestMatrix.CabRequests[msg.ButtonEvent.Floor] = true
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
				state.MasterStateStore.UpdateStatus(status)

			case message.MasterQuery:
				// Always respond with your known master (even if you're not master yourself)
				response := message.Message{
					Type:     message.MasterAnnouncement,
					MasterID: CurrentMasterID,
				}
				msgTx <- response

			case message.MasterAnnouncement:
				fmt.Println("[MH] Received MasterAnnouncement:", msg.MasterID)
				CurrentMasterID = msg.MasterID
				config.IsMaster = (config.ElevatorID == msg.MasterID)

				// Cancel election timer if waiting
				if MasterQueryTimer != nil {
					MasterQueryTimer.Stop()
					MasterQueryTimer = nil
				}

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

			if config.IsMaster {
				state.MasterStateStore.SetHallRequest(be)
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
	fmt.Printf("[MH] Message received from elevatorID: %d:\n\tType: %s, msgID: %s, sending ack\n", msg.ElevatorID, utils.MessageTypeToString(msg.Type), msg.MsgID)

	ackMsg := message.Message{
		Type:       message.Ack,
		ElevatorID: config.ElevatorID,
		//MsgID:      msgID.Get(),
		AckID: msg.MsgID,
	}

	msgTx <- ackMsg
}

// orderSenderWorker continuously listens for order requests and handles cancellation/resending.
func OrderSenderWorker(orderRequestCh <-chan HRA.OrderData, msgTx chan message.Message, trackerChan chan *message.AckTracker, msgID *message.MsgID) {
	var currentTracker *message.AckTracker

	resendTicker := time.NewTicker(config.ResendInterval)
	defer resendTicker.Stop()

	// Use a for-select loop that also listens for new orders.
	for {
		if config.IsMaster {
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

				fmt.Printf("[MH: Master] Staring sending of MsgID: %s\n", orderMsg.MsgID)

				timeoutTicker := time.NewTicker(config.MsgTimeout)
				defer timeoutTicker.Stop()

				// Enter a nested loop to resend the current order.
			resendLoop:
				for {
					select {
					case <-tracker.Done:
						fmt.Printf("[MH: Master] Terminating sending of MsgID: %s, stopping order broadcast\n", orderMsg.MsgID)
						break resendLoop

					case <-resendTicker.C:
						fmt.Println("[MH: Master] Resending order for MsgID:", orderMsg.MsgID)
						msgTx <- orderMsg

					case <-timeoutTicker.C:
						fmt.Println("[MH: Master] Timeout: Terminating sending of MsgID: %s", orderMsg.MsgID)
						break resendLoop

					case newReq := <-orderRequestCh:
						currentTracker.Terminate()

						req = newReq
						break resendLoop
					}
				}
			}
		}
	}
}


func InitMasterDiscovery(msgTx chan message.Message) {
	go func() {
		fmt.Println("[Startup] Sending MasterQuery")

		msgTx <- message.Message{
			Type:       message.MasterQuery,
			ElevatorID: config.ElevatorID,
		}

		MasterQuerySent = true
		MasterQueryTimer = time.NewTimer(750 * time.Millisecond)

		<-MasterQueryTimer.C

		fmt.Println("[Startup] No MasterAnnouncement received — starting heartbeat/election loop")
		go MonitorMasterHeartbeat(state.MasterStateStore, msgTx)
	}()
}

