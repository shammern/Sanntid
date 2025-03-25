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
		MsgID:       e.counter.Next(),
		ButtonEvent: event,
	}

	// For hall events, broadcast until all ACKs are received.
	//if event.Button != drivers.BT_Cab {
	expected := utils.GetActiveElevators()
	tracker := message.NewAckTracker(msg.MsgID, expected)

	tracker.ExpectedAcks[config.ElevatorID] = true

	// Register the tracker in the outstanding acks channel.
	e.ackTrackerChan <- tracker

	ticker := time.NewTicker(config.ResendInterval)
	defer ticker.Stop()

	for {
		select {
		case <-tracker.Done:
			//fmt.Printf("[ElevatorTransceiver] All ACKs received for MsgID: %s, stopping broadcast to master\n", tracker.MsgID)
			return
		case <-ticker.C:
			e.msgTx <- msg
		}
	}
	//}

	// For cab button events, simply send the message.
	//e.msgTx <- msg
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
