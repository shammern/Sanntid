package message

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/requestmatrix"
	"fmt"
	"sync"
	"time"
)

var MsgCounter = MsgID{}

type MessageType int

const (
	ElevatorStatus MessageType = iota
	ButtonEvent
	OrderDelegation
	CompletedOrder
	Ack
	Heartbeat
	MasterAnnouncement
	MasterQuery
	RecoveryState
	RecoveryQuery
)

type ElevatorState struct {
	ElevatorID      int
	State           int
	Available       bool
	Direction       int
	CurrentFloor    int
	TravelDirection int
	LastUpdated     time.Time
	RequestMatrix   requestmatrix.RequestMatrixDTO
	ErrorTrigger    int
}

type Message struct {
	Type         MessageType
	ElevatorID   int
	MsgID        string
	StateData    *ElevatorState
	ButtonEvent  drivers.ButtonEvent
	OrderData    map[string][][2]bool
	HallRequests [][2]bool
	AckID        string
	MasterID     int
}

type MsgID struct {
	mu       sync.Mutex
	senderID int
	id       int
}

func InitMsgCounter() {
	MsgCounter = MsgID{
		senderID: config.ElevatorID,
		id:       1,
	}
}

// Gets the next msgID in line and increments the counter
func (mc *MsgID) Next() string {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	currentID := mc.id
	mc.id++
	return fmt.Sprintf("%d-%d", mc.senderID, currentID)
}

func (m *MsgID) Get() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return fmt.Sprintf("%d-%d", m.senderID, m.id)
}
