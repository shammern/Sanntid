package message

import (
	"elevator-project/pkg/orders"
	"encoding/json"
	"time"
)

type MessageType int

const (
	State MessageType = iota
	Order
	Ack
	Heartbeat
	MasterSlaveConfig
)

type ElevatorState struct {
	ElevatorID    int                  `json:"elevatorID"`
	State         int                  `json:"state"`
	Direction     int                  `json:"direction"`
	CurrentFloor  int                  `json:"currentFloor"`
	TargetFloor   int                  `json:"targetFloor"`
	LastUpdated   time.Time            `json:"lastUpdated"`
	RequestMatrix orders.RequestMatrix `json:"requestMatrix"`
}

type OrderData struct {
	Floor      int    `json:"floor"`
	ButtonType string `json:"button_type"`
}

type Message struct {
	Type       MessageType    `json:"type"`
	ElevatorID int            `json:"elevator_id"`
	Seq        int            `json:"seq"`
	StateData  *ElevatorState `json:"state_data,omitempty"`
	OrderData  *OrderData     `json:"order_data,omitempty"`
	AckSeq     int            `json:"ack_seq,omitempty"`
}

func Marshal(msg Message) ([]byte, error) {
	return json.Marshal(msg)
}

func Unmarshal(data []byte) (Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return msg, err
}
