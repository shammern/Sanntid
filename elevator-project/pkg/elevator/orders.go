package elevator

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/message"
	"elevator-project/pkg/state"
	"elevator-project/pkg/utils"
	"fmt"
	"strconv"
	"time"
)

func (e *Elevator) ordersAbove() bool {
	cab := e.requestMatrix.GetCabRequest()
	hall := e.requestMatrix.GetHallRequest()
	for i := e.currentFloor + 1; i < len(cab); i++ {
		if cab[i] || hall[i][0] || hall[i][1] {
			return true
		}
	}
	return false
}

func (e *Elevator) ordersBelow() bool {
	cab := e.requestMatrix.GetCabRequest()
	hall := e.requestMatrix.GetHallRequest()
	for i := 0; i < e.currentFloor; i++ {
		if cab[i] || hall[i][0] || hall[i][1] {
			return true
		}
	}
	return false
}

func (e *Elevator) anyOrdersAtCurrentFloor() bool {
	cab := e.requestMatrix.GetCabRequest()
	hall := e.requestMatrix.GetHallRequest()
	if e.currentFloor < 0 || e.currentFloor >= len(cab) {
		return false
	}
	return cab[e.currentFloor] || hall[e.currentFloor][0] || hall[e.currentFloor][1]
}

func (e *Elevator) shouldStop() bool {
	cab := e.requestMatrix.GetCabRequest()
	hall := e.requestMatrix.GetHallRequest()
	if e.currentFloor < 0 || e.currentFloor >= len(cab) {
		return false
	}

	switch e.travelDirection {
	case Up:
		return hall[e.currentFloor][0] ||
			cab[e.currentFloor] ||
			!e.ordersAbove() ||
			e.currentFloor == 0 ||
			e.currentFloor == len(cab)-1
	case Down:
		return hall[e.currentFloor][1] ||
			cab[e.currentFloor] ||
			!e.ordersBelow() ||
			e.currentFloor == 0 ||
			e.currentFloor == len(cab)-1
	case Stop:
		return true
	default:
		return false
	}
}

// clearHallReqsAtFloor checks for active requests at the current floor and clears them using the request matrix methods.
// It also sends a CompletedOrder message via NotifyMaster.
func (e *Elevator) clearHallReqsAtFloor() {
	switch e.travelDirection {
	case Up:
		hall := e.requestMatrix.GetHallRequest()
		if hall[e.currentFloor][0] {
			e.requestMatrix.ClearHallRequest(e.currentFloor, 0)
			go e.NotifyMaster(message.CompletedOrder, drivers.ButtonEvent{
				Floor:  e.currentFloor,
				Button: drivers.BT_HallUp,
			})
			drivers.SetButtonLamp(drivers.BT_HallUp, e.currentFloor, false)
		} else if hall[e.currentFloor][1] {
			e.requestMatrix.ClearHallRequest(e.currentFloor, 1)
			e.NotifyMaster(message.CompletedOrder, drivers.ButtonEvent{
				Floor:  e.currentFloor,
				Button: drivers.BT_HallDown,
			})
			drivers.SetButtonLamp(drivers.BT_HallDown, e.currentFloor, false)
		}

		if e.requestMatrix.GetCabRequest()[e.currentFloor] {
			drivers.SetButtonLamp(drivers.BT_Cab, e.currentFloor, false)
			e.requestMatrix.ClearCabRequest(e.currentFloor)
			go e.NotifyMaster(message.CompletedOrder, drivers.ButtonEvent{
				Floor:  e.currentFloor,
				Button: drivers.BT_Cab,
			})
		}
	case Down:
		hall := e.requestMatrix.GetHallRequest()
		if hall[e.currentFloor][1] {
			e.requestMatrix.ClearHallRequest(e.currentFloor, 1)
			go e.NotifyMaster(message.CompletedOrder, drivers.ButtonEvent{
				Floor:  e.currentFloor,
				Button: drivers.BT_HallDown,
			})
			drivers.SetButtonLamp(drivers.BT_HallDown, e.currentFloor, false)
		} else if hall[e.currentFloor][0] {
			e.requestMatrix.ClearHallRequest(e.currentFloor, 0)
			go e.NotifyMaster(message.CompletedOrder, drivers.ButtonEvent{
				Floor:  e.currentFloor,
				Button: drivers.BT_HallUp,
			})
			drivers.SetButtonLamp(drivers.BT_HallUp, e.currentFloor, false)
		}

		if e.requestMatrix.GetCabRequest()[e.currentFloor] {
			drivers.SetButtonLamp(drivers.BT_Cab, e.currentFloor, false)
			e.requestMatrix.ClearCabRequest(e.currentFloor)
			go e.NotifyMaster(message.CompletedOrder, drivers.ButtonEvent{
				Floor:  e.currentFloor,
				Button: drivers.BT_Cab,
			})
		}
	case Stop:
		if e.requestMatrix.GetCabRequest()[e.currentFloor] {
			e.requestMatrix.ClearCabRequest(e.currentFloor)
			go e.NotifyMaster(message.CompletedOrder, drivers.ButtonEvent{
				Floor:  e.currentFloor,
				Button: drivers.BT_Cab,
			})
			drivers.SetButtonLamp(drivers.BT_Cab, e.currentFloor, false)
		}

		hall := e.requestMatrix.GetHallRequest()
		if hall[e.currentFloor][0] {
			e.requestMatrix.ClearHallRequest(e.currentFloor, 0)
			go e.NotifyMaster(message.CompletedOrder, drivers.ButtonEvent{
				Floor:  e.currentFloor,
				Button: drivers.BT_HallUp,
			})
			drivers.SetButtonLamp(drivers.BT_HallUp, e.currentFloor, false)
		} else if hall[e.currentFloor][1] {
			e.requestMatrix.ClearHallRequest(e.currentFloor, 1)
			go e.NotifyMaster(message.CompletedOrder, drivers.ButtonEvent{
				Floor:  e.currentFloor,
				Button: drivers.BT_HallDown,
			})
			drivers.SetButtonLamp(drivers.BT_HallDown, e.currentFloor, false)
		}
	}
}


func (e *Elevator) UpdateElevatorState(ev FsmEvent) {
	e.ch_fsmEvents <- ev
}

func (e *Elevator) SendOrderToFSM(order Order) {
	e.ch_orders <- order
}

// NotifyMaster sends a message to the master elevator.
// If the current instance is the master, it updates the MasterStore directly.
func (e *Elevator) NotifyMaster(msgType message.MessageType, event drivers.ButtonEvent) {
	// If we are running as master, update the MasterStore directly
	if config.IsMaster {
		fmt.Printf("[ElevatorTransceiver] Master handling local update for message type: %s, Floor: %d, Button: %s\n",
			utils.MessageTypeToString(msgType), event.Floor, utils.ButtonTypeToString(event.Button))

		if msgType == message.CompletedOrder {
			// Clear the order from the MasterStore.
			state.MasterStateStore.ClearOrder(event, config.ElevatorID)
			state.MasterStateStore.ClearHallRequest(event)

		}
	}

	// Non-master behavior: prepare and broadcast the message over the network.
	fmt.Printf("[ElevatorTransceiver] Sending MsgID: %s, type: %s, Floor: %d, Button: %s\n",
		message.MsgCounter.Get(), utils.MessageTypeToString(msgType), event.Floor, utils.ButtonTypeToString(event.Button))

	msg := message.Message{
		Type:        msgType,
		ElevatorID:  config.ElevatorID,
		MsgID:       e.msgCounter.Next(),
		ButtonEvent: event,
	}

	// For hall events, broadcast until all ACKs are received.
	//if event.Button != drivers.BT_Cab {
	expected := utils.GetActiveElevators()
	tracker := message.NewAckTracker(msg.MsgID, expected)

	tracker.SetOwnAck()

	// Register the tracker in the outstanding acks channel.
	e.ch_ackTracker <- tracker

	ticker := time.NewTicker(config.ResendInterval)
	defer ticker.Stop()

	timeout := time.After(config.MsgTimeout)

	for {
		select {
		case <-tracker.GetDoneChan():
			//fmt.Printf("[ElevatorTransceiver] All ACKs received for MsgID: %s, stopping broadcast to master\n", tracker.MsgID)
			return
		case <-ticker.C:
			e.ch_msgTx <- msg

		case <-timeout:
			fmt.Println("[ElevatorFSM] Message timed out, stopping sending")
			return
		}
	}
}

// Takes a ordermap and extract the orders for the currentelevator. Orders are packed into a list and returned.
func ConvertOrderDataToOrders(orderData map[string][][2]bool) []Order {
	var ordersList []Order
	orders := orderData[strconv.Itoa(config.ElevatorID)]

	for floor, calls := range orders {
		// Hall up call
		if calls[0] {
			ordersList = append(ordersList, Order{
				Event: drivers.ButtonEvent{Floor: floor, Button: drivers.BT_HallUp},
				Flag:  true,
			})
		} else {
			ordersList = append(ordersList, Order{
				Event: drivers.ButtonEvent{Floor: floor, Button: drivers.BT_HallUp},
				Flag:  false,
			})
		}
		// Hall down call
		if calls[1] {
			ordersList = append(ordersList, Order{
				Event: drivers.ButtonEvent{Floor: floor, Button: drivers.BT_HallDown},
				Flag:  true,
			})
		} else {
			ordersList = append(ordersList, Order{
				Event: drivers.ButtonEvent{Floor: floor, Button: drivers.BT_HallDown},
				Flag:  false,
			})
		}
	}

	return ordersList
}
