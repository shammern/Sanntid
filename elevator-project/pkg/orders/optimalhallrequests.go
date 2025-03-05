package orders

import (
	"elevator-project/pkg/drivers"
	"math"
)

const (
	travelTime   = 2500
	doorOpenTime = 3000
)

type Dirn int

type ElevatorState struct {
	RequestMatrix *RequestMatrix
	Floor         int
	Direction     Dirn
}

type Req struct {
	Active     bool
	AssignedTo string
}

type State struct {
	ID    string
	State ElevatorState
	Time  int64
}


func isUnassigned(r Req) bool {
	return r.Active && r.AssignedTo == ""
}


func anyUnassigned(reqs [][]Req) bool {
	for _, floor := range reqs {
		for _, r := range floor {
			if isUnassigned(r) {
				return true
			}
		}
	}
	return false
}

func clearRequestsAtFloor(e SimulatedElevator) SimulatedElevator {
	e.RequestMat.CabRequests[e.Floor] = false 

	switch e.Direction {
	case drivers.MD_Up:
		e.RequestMat.HallRequests[e.Floor][0] = false 
	case drivers.MD_Down:
		e.RequestMat.HallRequests[e.Floor][1] = false 
	case drivers.MD_Stop:
		e.RequestMat.HallRequests[e.Floor][0] = false
		e.RequestMat.HallRequests[e.Floor][1] = false
	}

	return e
}

func requestsAbove(e ElevatorState) bool {
	for i := e.Floor + 1; i < len(e.RequestMatrix.HallRequests); i++ {
		for _, req := range e.RequestMatrix.HallRequests[i] {
			if req {
				return true
			}
		}
		if e.RequestMatrix.CabRequests[i] {
			return true
		}
	}
	return false 
}


func requestsBelow(e ElevatorState) bool {
	for i := 0; i < e.Floor; i++ {
		for _, req := range e.RequestMatrix.HallRequests[i] {
			if req {
				return true
			}
		}
		if e.RequestMatrix.CabRequests[i] {
			return true
		}
	}
	return false
}

func anyRequestsAtFloor(e ElevatorState) bool {
	for _, req := range e.RequestMatrix.HallRequests[e.Floor] {
		if req {
			return true
		}
	}
	return e.RequestMatrix.CabRequests[e.Floor]
}

func shouldStop(e SimulatedElevator) bool {
	switch e.Direction {
	case drivers.MD_Up:
		return e.RequestMat.HallRequests[e.Floor][0] || e.RequestMat.CabRequests[e.Floor]
	case drivers.MD_Down:
		return e.RequestMat.HallRequests[e.Floor][1] || e.RequestMat.CabRequests[e.Floor]
	default:
		return true
	}
}

func chooseDirection(e ElevatorState) Dirn {
	if requestsAbove(e) {
		return 1
	} else if anyRequestsAtFloor(e) {
		return 0
	} else if requestsBelow(e) {
		return -1
	}
	return 0
}

type SimulatedElevator struct {
	Floor      int
	Direction  drivers.MotorDirection
	RequestMat *RequestMatrix
}

func CalculateTimeToIdle(e SimulatedElevator) int {
	duration := 0

	switch e.Direction {
	case drivers.MD_Stop:
		e.Direction = drivers.MotorDirection(chooseDirection(ElevatorState{RequestMatrix: e.RequestMat, Floor: e.Floor, Direction: Dirn(e.Direction)}))
		if e.Direction == drivers.MD_Stop {
			return duration
		}
	case drivers.MD_Up, drivers.MD_Down:
		duration += travelTime / 2
		e.Floor += int(e.Direction)
	}

	for {
		if shouldStop(e) {
			e = clearRequestsAtFloor(e)
			duration += doorOpenTime

			e.Direction = drivers.MotorDirection(chooseDirection(ElevatorState{
				RequestMatrix: e.RequestMat, Floor: e.Floor, Direction: Dirn(e.Direction),
			}))
			if e.Direction == drivers.MD_Stop {
				return duration
			}
		}
		e.Floor += int(e.Direction)
		duration += travelTime
	}
}

//Finds optimal elevator to handle a new order based on simulation of time it will take to complete the new order
func AssignElevator(requestFloor int, requestButton drivers.ButtonType, elevators map[string]SimulatedElevator) string {
	bestElevator := ""
	bestTime := math.MaxInt32

	for id, e := range elevators {
		copyE := e                                                              
		copyE.RequestMat.SetHallRequest(requestFloor, int(requestButton), true) 
		timeToIdle := CalculateTimeToIdle(copyE)                               

		if timeToIdle < bestTime {
			bestTime = timeToIdle
			bestElevator = id
		}
	}
	return bestElevator
}
