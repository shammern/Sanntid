package elevator

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/message"
	"elevator-project/pkg/utils"

	"fmt"
	"time"
)

// Broadcasts a message on the network until master acks the message. Sends either a buttonEvent og OrderCompleted msg type
func (e *Elevator) NotifyMaster(msgType message.MessageType, event drivers.ButtonEvent) {
	fmt.Printf("[ElevatorTransceiver] Sending messagetype: %s, Floor: %d, Button: %s\n",
		utils.MessageTypeToString(msgType), event.Floor, utils.ButtonTypeToString(event.Button))

	msg := message.Message{
		Type:        msgType,
		ElevatorID:  e.ElevatorID,
		MsgID:       e.counter.Next(),
		ButtonEvent: event,
	}

	if event.Button != drivers.BT_Cab {

		expected := utils.GetActiveElevators()
		tracker := message.NewAckTracker(msg.MsgID, expected)

		tracker.ExpectedAcks[config.ElevatorID] = true

		// Register the tracker in the global outstandingAcks.
		e.ackTrackerChan <- tracker

		ticker := time.NewTicker(config.ResendInterval)
		defer ticker.Stop()

		for {
			select {
			case <-tracker.Done:
				fmt.Printf("[ElevatorTransceiver] All acks received for MsgID: %d, stopping broadcast to master\n", tracker.MsgID)
				//TODO: Delete tracker
				return
			case <-ticker.C:
				e.msgTx <- msg
			}
		}
	}

	e.msgTx <- msg
}
