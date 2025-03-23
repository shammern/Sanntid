package config

import "time"

var ElevatorAddresses = map[int]string{
	1: "localhost:7331",
	2: "localhost:7332",
	3: "localhost:7333",
}

var UDPAddresses = map[int]string{
	1: "127.0.0.1:8001",
	2: "127.0.0.1:8002",
	3: "127.0.0.1:8003",
}

var UDPAckAddresses = map[int]string{
	1: "127.0.0.1:8011",
	2: "127.0.0.1:8012",
	3: "127.0.0.1:8013",
}

var NumFloors = 4
var ElevatorID = 0
var HeartBeatInterval = 100 * time.Millisecond
var WorldviewBCInterval = 100 * time.Millisecond
var ResendInterval = 10 * time.Millisecond
var BCport = 15500
var P2Pport = 16000
var IsMaster = false
