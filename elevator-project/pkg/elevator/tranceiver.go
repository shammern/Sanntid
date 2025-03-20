package elevator

import (
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/message"
	"elevator-project/pkg/utils"

	"fmt"
	"time"
)

//Broadcasts a message on the network until master acks the message. Sends either a buttonEvent og OrderCompleted msg type
//Currently blocking as it waits for ack.
//TODO: Implement a nonblocking way that can handle sending multiple messages and filter out correct ack message. Maybe implement a new ackChannel everytime?
func (e *Elevator) NotifyMaster(msgType message.MessageType, event drivers.ButtonEvent) {
	fmt.Printf("[ElevatorTranceiver] Sending messagetype: %s, Floor: %d, Type: %s\n", utils.MessageTypeToString(msgType), e.currentFloor, utils.ButtonTypeToString(event.Button))
	msg := message.Message{
		Type:        msgType,
		ElevatorID:  e.ElevatorID,
		MsgID:       e.counter.Next(),
		ButtonEvent: event,
	}

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	if event.Button != drivers.BT_Cab {
		for {
			select {
			// If an acknowledgement is received, break out of the loop.
			case <-e.ackChan:
				return
			// Otherwise, on each tick, send the message.
			case <-ticker.C:
			
				e.msgTx <- msg
			}
		}
	}

	//If cab button only sends message once
	e.msgTx <- msg
}


