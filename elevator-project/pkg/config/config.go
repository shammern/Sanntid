package config

import "time"

var ElevatorAddresses = map[int]string{
	1: "localhost:7331",
	2: "localhost:7332",
	3: "localhost:7333",
}

const (
	HeartBeatInterval          = 5 * time.Millisecond
	WorldviewBCInterval        = 100 * time.Millisecond
	ResendInterval             = 10 * time.Millisecond
	Timeout                    = 500 * time.Millisecond
	QueryMasterTimer           = 500 * time.Millisecond
	MsgTimeout                 = 2 * time.Second
	TimeBetweenFloorsThreshold = 5 * time.Second
	DoorOpenThreshold          = 5 * time.Second
	MasterTimeout              = 2 * time.Second

	BCport  = 15500
	P2Pport = 16000

	NumFloors = 4
)

var IsMaster = false
var ElevatorID = 0
