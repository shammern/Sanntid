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

const HeartBeatInterval = 5 * time.Millisecond
const WorldviewBCInterval = 100 * time.Millisecond
const ResendInterval = 10 * time.Millisecond
const Timeout = 500 * time.Millisecond
const QueryMasterTimer = 500 * time.Millisecond
const MsgTimeout = 2 * time.Second
const TimeBetweenFloorsThreshold = 5 * time.Second
const DoorOpenThreshold = 10 * time.Second

const BCport = 15500
const P2Pport = 16000

const NumFloors = 4

var IsMaster = false
var ElevatorID = 0
