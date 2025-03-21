package message

import (
	RM "elevator-project/pkg/RequestMatrix"
	"elevator-project/pkg/drivers"
	"fmt"
	"sync"
	"time"
)

type MessageType int

const (
	State           MessageType = iota // Full worldview
	ButtonEvent                        // All types of buttonpresses
	OrderDelegation                    // Master delegates an order to a specific elevator
	CompletedOrder
	Ack
	Heartbeat
	MasterSlaveConfig // ???
	Promotion         // Promotion msg letting other elevators know that a new elevator is master?
)

type ElevatorState struct {
	ElevatorID      int
	State           int
	Direction       int
	CurrentFloor    int
	TravelDirection int
	LastUpdated     time.Time
	RequestMatrix   RM.RequestMatrix
}

type Message struct {
	Type         MessageType
	ElevatorID   int
	MsgID        int
	StateData    *ElevatorState //Why is this a pointer?
	ButtonEvent  drivers.ButtonEvent
	OrderData    map[string][][2]bool //Hallorders for individual elevators
	HallRequests [][2]bool            // All active hallorders aka the halllights
	AckID        int                  //AckID = msgID for the corresponding message requiring an ack
}

type MsgID struct {
	mu sync.Mutex
	id int
}

func (mc *MsgID) Next() int {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	currentID := mc.id
	mc.id++
	return currentID
}

func (m *MsgID) Get() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.id
}

type AckTracker struct {
	MsgID        int           // The unique identifier for the message.
	SentTime     time.Time     // Timestamp when the message was sent.
	ExpectedAcks map[int]bool  // Map of elevator IDs to whether their ack has been received.
	Done         chan struct{} // Channel to signal when all acks are received.
}

type OutstandingAcks struct {
	Tracker        map[int]*AckTracker
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

func NewAckTracker(msgID int, expected []int) *AckTracker {
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
		Tracker:        make(map[int]*AckTracker),
		AckTrackerChan: trackChan,
		AckChan:        ackChan,
	}
}

func (oa *OutstandingAcks) RegisterAckTracker(tracker *AckTracker) {
	oa.Tracker[tracker.MsgID] = tracker
}

// DeleteAckTracker removes an AckTracker from the global map using its message ID.
func (oa *OutstandingAcks) DeleteAckTracker(msgID int) {
	delete(oa.Tracker, msgID)
}

// RunAckTracker continuously monitors for new AckTrackers or acks and updates the trackers.
func (oa *OutstandingAcks) RunAckMonitor() {
	for {
		select {

		// When a new tracker arrives, register it.
		case tracker := <-oa.AckTrackerChan:
			fmt.Println("[AckTracker] Received an new Acktracker")
			oa.RegisterAckTracker(tracker)

		// When an ack message arrives, update the corresponding tracker.
		case ack := <-oa.AckChan:
			if tracker, exists := oa.Tracker[ack.AckID]; exists {
				fmt.Println("[AckTracker] Received an ACK for an active message")

				tracker.ExpectedAcks[ack.ElevatorID] = true

				if tracker.AllAcked() {
					close(tracker.Done)
					oa.DeleteAckTracker(ack.AckID)
				}
			}
		}
	}
}
