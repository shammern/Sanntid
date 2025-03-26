package app

import (
	"elevator-project/pkg/HRA"
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/elevator"
	"elevator-project/pkg/master"
	"elevator-project/pkg/message"
	"elevator-project/pkg/network/peers"
	"elevator-project/pkg/systemdata"
	"elevator-project/pkg/utils"
	"fmt"
	"time"
)

// Messagehandler receives all messages from network and routes them based on messagetype
func MessageHandler(msgRx chan message.Message, ackChan chan message.Message, msgTx chan message.Message, elevatorFSM *elevator.Elevator, ch_ackTracker chan *message.AckTracker, masterAnnounced chan struct{}) {
	for msg := range msgRx {
		//Filter out own messages unless they are of type RecoveryState
		if msg.Type != message.RecoveryState && msg.ElevatorID == config.ElevatorID {
			continue
		}

		switch msg.Type {
		case message.Ack:
			ackChan <- msg

		case message.OrderDelegation:
			SendAck(msg, msgTx)
			//Extracts OrderData and send them to elevator
			orderData := msg.OrderData

			orders := elevator.ConvertOrderDataToOrders(orderData)
			for _, order := range orders {
				elevatorFSM.SendOrderToFSM(order)
			}

			systemdata.MasterStateStore.SetHallRequests(msg.HallRequests)
			elevatorFSM.SetHallLigths(systemdata.MasterStateStore.HallRequests)

		case message.CompletedOrder:
			SendAck(msg, msgTx)

			//Updates systemdata and updates light status
			fmt.Printf("[MH] Order has been completed: ElevatorID: %d, Floor: %d, ButtonType: %s\n", msg.ElevatorID, msg.ButtonEvent.Floor, utils.ButtonTypeToString(msg.ButtonEvent.Button))
			systemdata.MasterStateStore.ClearOrderFromElevator(msg.ButtonEvent, msg.ElevatorID)
			systemdata.MasterStateStore.ClearHallRequest(msg.ButtonEvent)
			elevatorFSM.SetHallLigths(systemdata.MasterStateStore.HallRequests)

		case message.ButtonEvent:

			if config.IsMaster {
				switch msg.ButtonEvent.Button {
				case drivers.BT_HallDown, drivers.BT_HallUp:
					if !systemdata.MasterStateStore.GetHallRequests()[msg.ButtonEvent.Floor][int(msg.ButtonEvent.Button)] {
						systemdata.MasterStateStore.SetHallRequest(msg.ButtonEvent)
					}

				case drivers.BT_Cab:
					if !systemdata.MasterStateStore.Elevators[msg.ElevatorID].RequestMatrix.CabRequests[msg.ButtonEvent.Floor] {
						systemdata.MasterStateStore.Elevators[msg.ElevatorID].RequestMatrix.CabRequests[msg.ButtonEvent.Floor] = true
					}

				}
			}
			SendAck(msg, msgTx)

		case message.ElevatorStatus:
			status := systemdata.ElevatorStatus{
				ElevatorID:      msg.ElevatorID,
				State:           msg.StateData.State,
				CurrentFloor:    msg.StateData.CurrentFloor,
				TravelDirection: msg.StateData.TravelDirection,
				RequestMatrix:   msg.StateData.RequestMatrix,
				LastUpdated:     msg.StateData.LastUpdated,
				Available:       msg.StateData.Available,
				ErrorTrigger:    msg.StateData.ErrorTrigger,
			}
			systemdata.MasterStateStore.UpdateElevatorStatus(status)

			if msg.ElevatorID == config.CurrentMasterID {
				systemdata.MasterStateStore.HallRequests = msg.HallRequests
			}

		case message.MasterQuery:
			go master.BroadcastMasterAnnouncement(msgTx, config.CurrentMasterID, ch_ackTracker)

		case message.MasterAnnouncement:
			if msg.MasterID != config.CurrentMasterID {
				fmt.Println("[MH] Received MasterAnnouncement:", msg.MasterID)
				config.CurrentMasterID = msg.MasterID
				config.IsMaster = (config.ElevatorID == config.CurrentMasterID)

				select {
				case masterAnnounced <- struct{}{}:
				default:
				}
			}

		case message.RecoveryState:
			if msg.ElevatorID == config.ElevatorID {
				ackChan <- message.Message{
					ElevatorID: config.ElevatorID,
					AckID:      fmt.Sprintf("%d-%d", config.ElevatorID, 0),
				}
				fmt.Printf("[MH] RecoverState received from master\n")
				elevatorFSM.RecoverState(msg.StateData)
			}

		case message.RecoveryQuery:
			if config.IsMaster {
				var recoverStateData *message.ElevatorState
				if status, exists := systemdata.MasterStateStore.Elevators[msg.ElevatorID]; exists {
					// If a record exists, include the stored cab orders
					recoverStateData = &message.ElevatorState{
						ElevatorID:      status.ElevatorID,
						State:           status.State,
						CurrentFloor:    status.CurrentFloor,
						TravelDirection: status.TravelDirection,
						ErrorTrigger:    status.ErrorTrigger,
						RequestMatrix:   status.RequestMatrix,
					}

					recoverMsg := message.Message{
						Type:       message.RecoveryState,
						ElevatorID: msg.ElevatorID,
						StateData:  recoverStateData,
					}

					msgTx <- recoverMsg

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
		systemdata.MasterStateStore.UpdateElevatorStatus(status)
		stateMsg := message.Message{
			Type:         message.ElevatorStatus,
			ElevatorID:   status.ElevatorID,
			HallRequests: systemdata.MasterStateStore.HallRequests,

			StateData: &message.ElevatorState{
				ElevatorID:      status.ElevatorID,
				State:           status.State,
				Available:       status.Available,
				CurrentFloor:    status.CurrentFloor,
				TravelDirection: status.TravelDirection,
				LastUpdated:     time.Now(),
				RequestMatrix:   status.RequestMatrix,
				ErrorTrigger:    status.ErrorTrigger,
			},
		}

		msgTx <- stateMsg
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
				if len(peers.LatestPeerUpdate.Peers) > 1 {
					systemdata.MasterStateStore.SetHallRequest(be)
				} else {
					fmt.Println("[ERROR] Elevator is offline and wont service hallcalls")
				}
			} else {
				elevatorFSM.NotifyMaster(message.ButtonEvent, be)
			}

			if be.Button == drivers.BT_Cab {
				order := elevator.Order{
					Event: be,
					Flag:  true,
				}

				elevatorFSM.SendOrderToFSM(order)
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
			//No stop logic has been implemented.
		}
	}
}

func SendAck(msg message.Message, msgTx chan message.Message) {
	fmt.Printf("[MH] Message received from elevatorID: %d:\n\tType: %s, msgID: %s, sending ack\n", msg.ElevatorID, utils.MessageTypeToString(msg.Type), msg.MsgID)

	ackMsg := message.Message{
		Type:       message.Ack,
		ElevatorID: config.ElevatorID,
		AckID:      msg.MsgID,
	}

	msgTx <- ackMsg
}

// orderSenderWorker continuously listens for order requests and handles cancellation/resending.
func OrderSenderWorker(orderRequestCh <-chan HRA.Output, msgTx chan message.Message, trackerChan chan *message.AckTracker, msgID *message.MsgID) {
	var currentTracker *message.AckTracker

	resendTicker := time.NewTicker(config.ResendInterval)
	defer resendTicker.Stop()

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
				tracker.SetOwnAck()
				trackerChan <- tracker
				currentTracker = tracker

				// Prepare the message that will be repeatedly sent.
				orderMsg := message.Message{
					Type:         message.OrderDelegation,
					ElevatorID:   config.ElevatorID,
					MsgID:        msgID.Next(),
					OrderData:    req.Orders,
					HallRequests: systemdata.MasterStateStore.HallRequests,
				}

				fmt.Printf("[MH: Master] Staring sending of MsgID: %s\n", orderMsg.MsgID)

				timeoutTicker := time.NewTicker(config.MsgTimeout)
				defer timeoutTicker.Stop()

				// Enter a nested loop to resend the current order.
			resendLoop:
				for {
					select {
					case <-tracker.GetDoneChan():
						fmt.Printf("[MH: Master] Terminating sending of MsgID: %s, stopping order broadcast\n", orderMsg.MsgID)
						break resendLoop

					case <-resendTicker.C:
						fmt.Println("[MH: Master] Resending order for MsgID: ", orderMsg.MsgID)
						msgTx <- orderMsg

					case <-timeoutTicker.C:
						fmt.Println("[MH: Master] Timeout: Terminating sending of MsgID: ", orderMsg.MsgID)
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
