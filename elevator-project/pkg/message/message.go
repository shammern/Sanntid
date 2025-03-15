package message

import (
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/orders"
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
	MasterAnnouncement // Ny meldingstype for å annonsere hvem som er master
	MasterQuery        // Melding for å spørre "Hvem er master?"
)

type ElevatorState struct {
	ElevatorID    int
	State         int
	Direction     int
	CurrentFloor  int
	TargetFloor   int
	LastUpdated   time.Time
	RequestMatrix orders.RequestMatrix
}

type Message struct {
	Type       MessageType
	ElevatorID int
	MsgID      int
	StateData  *ElevatorState
	OrderData  drivers.ButtonEvent
	AckID      int //AckID = msgID for the corresponding message requiring an ack
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
