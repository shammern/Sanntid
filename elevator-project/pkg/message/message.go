package message

import (
	RM "elevator-project/pkg/RequestMatrix"
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/network/peers"
	"fmt"
	"sync"
	"time"
)

var MsgCounter = MsgID{
	elevatorID: config.ElevatorID,
	id:         1,
}

type MessageType int

const (
	State           MessageType = iota // Full worldview
	ButtonEvent                        // All types of buttonpresses
	OrderDelegation                    // Master delegates an order to a specific elevator
	CompletedOrder
	Ack
	Heartbeat
	MasterAnnouncement // Ny meldingstype for å annonsere hvem som er master
	MasterQuery        // Melding for å spørre "Hvem er master?"
)

type ElevatorState struct {
	ElevatorID      int
	State           int
	Available       bool
	Direction       int
	CurrentFloor    int
	TravelDirection int
	LastUpdated     time.Time
	RequestMatrix   RM.RequestMatrix
}

type Message struct {
	Type         MessageType
	ElevatorID   int
	MsgID        string
	StateData    *ElevatorState //Why is this a pointer?
	ButtonEvent  drivers.ButtonEvent
	OrderData    map[string][][2]bool //Hallorders for individual elevators
	HallRequests [][2]bool            // All active hallorders aka the halllights
	AckID        string               //AckID = msgID for the corresponding message requiring an ack
	MasterID     int                  //Current amster
}

type MsgID struct {
	mu         sync.Mutex
	elevatorID int
	id         int
}

func InitMsgCounter(elevatorID int) {
	MsgCounter = MsgID{
		elevatorID: elevatorID,
		id:         1,
	}
}

// Next returns a composite message identifier in the form "elevatorID-counter".
func (mc *MsgID) Next() string {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	currentID := mc.id
	mc.id++
	return fmt.Sprintf("%d-%d", mc.elevatorID, currentID)
}

// Get returns the current composite message id without incrementing.
func (m *MsgID) Get() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return fmt.Sprintf("%d-%d", m.elevatorID, m.id)
}

func NewMsgId() *MsgID {
	return &MsgID{
		elevatorID: config.ElevatorID,
		id:         1,
	}
}

type AckTracker struct {
	MsgID        string        // The unique identifier for the message.
	SentTime     time.Time     // Timestamp when the message was sent.
	ExpectedAcks map[int]bool  // Map of elevator IDs to whether their ack has been received.
	Done         chan struct{} // Channel to signal when all acks are received.
	closeOnce    sync.Once
}

type OutstandingAcks struct {
	Tracker        map[string]*AckTracker
	AckTrackerChan chan *AckTracker
	AckChan        chan Message
}

// Checks if all expected acks have been received.
func (a *AckTracker) AllAcked() bool {
	for _, ack := range a.ExpectedAcks {
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
		MsgID:        msgID,
		SentTime:     time.Now(),
		ExpectedAcks: expectedAcks,
		Done:         make(chan struct{}),
	}
}

// NewOutstandingAcks returns a new OutstandingAcks structure.
func NewAckMonitor(trackChan chan *AckTracker, ackChan chan Message) *OutstandingAcks {
	return &OutstandingAcks{
		Tracker:        make(map[string]*AckTracker),
		AckTrackerChan: trackChan,
		AckChan:        ackChan,
	}
}

func (oa *OutstandingAcks) RegisterAckTracker(tracker *AckTracker) {
	oa.Tracker[tracker.MsgID] = tracker
}

// DeleteAckTracker removes an AckTracker from the global map using its message ID.
func (oa *OutstandingAcks) DeleteAckTracker(msgID string) {
	delete(oa.Tracker, msgID)
}

// Call this method to safely signal termination (for either ack completion or cancellation)
func (at *AckTracker) Terminate() {
	at.closeOnce.Do(func() {
		close(at.Done)
	})
}

// processAck processes an ack message: it logs which elevator sent the ack,
// updates the tracker, prints the list of pending acks, and if complete,
// closes the tracker and removes it from the global map.
func (oa *OutstandingAcks) processAck(tracker *AckTracker, ack Message) {

	//fmt.Printf("[AckTracker] Received an ACK from elevator %d for message %s\n", ack.ElevatorID, ack.AckID)

	// Update the tracker to indicate that own elevator has seen the msg
	tracker.ExpectedAcks[ack.ElevatorID] = true

	// Collect a list of elevators for which an ack is still pending.
	var pending []int
	for elevatorID, acked := range tracker.ExpectedAcks {
		if !acked {
			pending = append(pending, elevatorID)
		}
	}

	if len(pending) > 0 {
		fmt.Printf("[AckTracker] Still waiting for ACKs from elevators: %v for message: %s\n", pending, tracker.MsgID)
	} else {
		fmt.Printf("[AckTracker] All ACKs received for message %s\n", ack.AckID)
		tracker.Terminate()
		oa.DeleteAckTracker(ack.AckID)
	}
}

// RunAckMonitor continuously monitors for new AckTrackers or ack messages.
func (oa *OutstandingAcks) RunAckMonitor() {
	for {
		select {
		// When a new tracker arrives, register it.
		case tracker := <-oa.AckTrackerChan:
			//fmt.Println("[AckTracker] Received a new AckTracker")
			oa.RegisterAckTracker(tracker)

		// When an ack message arrives, update the corresponding tracker.
		case ack := <-oa.AckChan:
			if tracker, exists := oa.Tracker[ack.AckID]; exists {
				oa.processAck(tracker, ack)
			} else {
				// Log the case where an ack is received for an unknown message.
				//fmt.Printf("[AckTracker] Received an ACK for unknown message id %s from elevator %d\n", ack.AckID, ack.ElevatorID)
			}

		//If elevator go offline while we wait for ack this statement will clear their acks for the expected ack in all valid acktrackers
		case lostIDs := <-peers.PeerLostCh:
			for _, tracker := range oa.Tracker {
				for _, lostID := range lostIDs {
					if acked, exists := tracker.ExpectedAcks[lostID]; exists && !acked {
						tracker.ExpectedAcks[lostID] = true
						fmt.Printf("[AckTracker] Marking lost elevator %d as acked for message %s\n", lostID, tracker.MsgID)
					}
				}

				if tracker.AllAcked() {
					tracker.Terminate()
					oa.DeleteAckTracker(tracker.MsgID)
				}
			}
		}
	}
}
