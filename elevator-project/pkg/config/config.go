package config

import "time"

var ElevatorAddresses = map[int]string{
	1: "localhost:7331",
	2: "localhost:7332",
	3: "localhost:7333",
}

const HeartBeatInterval = 5 * time.Millisecond
const WorldviewBCInterval = 100 * time.Millisecond
const ResendInterval = 10 * time.Millisecond
const Timeout = 500 * time.Millisecond
const QueryMasterTimer = 500 * time.Millisecond
const MsgTimeout = 2 * time.Second
const TimeBetweenFloorsThreshold = 5 * time.Second
const DoorOpenThreshold = 10 * time.Second
const MasterTimeout = 2 * time.Second

const BCport = 15500
const P2Pport = 16000

const NumFloors = 4

var IsMaster = false
var ElevatorID = 0
