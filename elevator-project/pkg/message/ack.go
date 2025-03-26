package message

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/network/peers"
	"fmt"
	"sync"
)

// AckTracker keeps track of which units we require an ack from for a given message
type AckTracker struct {
	msgID        string
	expectedAcks map[int]bool
	done         chan struct{}
	closeOnce    sync.Once
}

type AckMonitor struct {
	tracker       map[string]*AckTracker
	ch_ackTracker chan *AckTracker
	ch_ack        chan Message
}

func (a *AckTracker) AllAcked() bool {
	for _, ack := range a.expectedAcks {
		if !ack {
			return false
		}
	}
	return true
}

func NewAckTracker(msgID string, expected []int) *AckTracker {
	expectedAcks := make(map[int]bool)
	for _, id := range expected {
		expectedAcks[id] = false
	}
	return &AckTracker{
		msgID:        msgID,
		expectedAcks: expectedAcks,
		done:         make(chan struct{}),
	}
}

func NewAckMonitor(ch_track chan *AckTracker, ch_ack chan Message) *AckMonitor {
	return &AckMonitor{
		tracker:       make(map[string]*AckTracker),
		ch_ackTracker: ch_track,
		ch_ack:        ch_ack,
	}
}

func (am *AckMonitor) RunAckMonitor() {
	for {
		select {
		// Register ackTracker into ackMonitor
		case tracker := <-am.ch_ackTracker:
			am.RegisterAckTracker(tracker)

		// Update ackTracker
		case ack := <-am.ch_ack:
			if tracker, exists := am.tracker[ack.AckID]; exists {
				am.processAck(tracker, ack)
			}

		// Clears active expextedAcks from elevators that goes offline
		case lostIDs := <-peers.Ch_peerLost:
			for _, tracker := range am.tracker {
				for _, lostID := range lostIDs {
					if acked, exists := tracker.expectedAcks[lostID]; exists && !acked {
						tracker.expectedAcks[lostID] = true
						fmt.Printf("[AckTracker] Marking lost elevator %d as acked for message %s\n", lostID, tracker.msgID)
					}
				}

				if tracker.AllAcked() {
					tracker.Terminate()
					am.DeleteAckTracker(tracker.msgID)
				}
			}
		}
	}
}

func (am *AckMonitor) RegisterAckTracker(tracker *AckTracker) {
	am.tracker[tracker.msgID] = tracker
}

func (am *AckMonitor) DeleteAckTracker(msgID string) {
	delete(am.tracker, msgID)
}

func (at *AckTracker) Terminate() {
	at.closeOnce.Do(func() {
		close(at.done)
	})
}

func (am *AckMonitor) processAck(tracker *AckTracker, ack Message) {

	fmt.Printf("[AckTracker] Received an ACK from elevator %d for message %s\n", ack.ElevatorID, ack.AckID)
	tracker.expectedAcks[ack.ElevatorID] = true

	// Collect a list of elevators for which an ack is still pending.
	var pending []int
	for elevatorID, acked := range tracker.expectedAcks {
		if !acked {
			fmt.Println("Marking as acked")
			pending = append(pending, elevatorID)
		}
	}

	// If all expectedAcks have been received, terminates the tracker and deletes from AckMonitor
	if len(pending) > 0 {
		fmt.Printf("[AckTracker] Still waiting for ACKs from elevators: %v for message: %s\n", pending, tracker.msgID)
	} else {
		fmt.Printf("[AckTracker] All ACKs received for message %s\n", ack.AckID)
		tracker.Terminate()
		am.DeleteAckTracker(ack.AckID)
	}
}

func (t *AckTracker) SetOwnAck() {
	t.expectedAcks[config.ElevatorID] = true
}

func (t *AckTracker) GetDoneChan() chan struct{} {
	return t.done
}
