package elevator

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/message"
	"elevator-project/pkg/requestmatrix"
	"elevator-project/pkg/systemdata"
	"fmt"
	"time"
)

type ElevatorState int

const (
	Init ElevatorState = iota
	Idle
	MovingUp
	MovingDown
	DoorOpen
	Error
)

type FsmEvent int

const (
	EventArrivedAtFloor FsmEvent = iota
	EventDoorTimerElapsed
	EventDoorObstructed
	EventDoorReleased
	EventSetError
)

type Direction int

const (
	Up   Direction = 1
	Down           = -1
	Stop           = 0
)

type ErrorType int

const (
	ErrorNone ErrorType = iota
	ErrorDoorTimeout
	ErrorMotorFailure
)

type Order struct {
	Event drivers.ButtonEvent
	Flag  bool
}

type Elevator struct {
	ElevatorID      int
	state           ElevatorState
	currentFloor    int
	travelDirection Direction
	doorObstructed  bool
	errorTrigger    ErrorType

	requestMatrix *requestmatrix.RequestMatrix
	msgCounter    *message.MsgID

	ch_orders     chan Order
	ch_fsmEvents  chan FsmEvent
	ch_msgTx      chan message.Message
	ch_ackTracker chan *message.AckTracker

	doorTimer           *time.Timer
	doorOpenStartTime   time.Time
	moveStartTime       time.Time
	lastRecoveryAttempt time.Time
}

func NewElevator(ElevatorID int, ch_msgTx chan message.Message, counter *message.MsgID, ch_trackerChan chan *message.AckTracker) *Elevator {
	return &Elevator{
		ElevatorID:      ElevatorID,
		ch_orders:       make(chan Order, 10),
		ch_fsmEvents:    make(chan FsmEvent, 10),
		ch_msgTx:        ch_msgTx,
		msgCounter:      counter,
		travelDirection: Stop,
		ch_ackTracker:   ch_trackerChan,
		requestMatrix:   requestmatrix.NewRequestMatrix(config.NumFloors),
		doorObstructed:  false,
	}
}

func (e *Elevator) InitElevator() {
	resendTicker := time.NewTicker(config.ResendInterval)
	defer resendTicker.Stop()

	timeout := time.After(config.ElevatorInitTimeout)

	recoveryMsg := message.Message{
		Type:       message.RecoveryQuery,
		MsgID:      fmt.Sprintf("%d-%d", config.ElevatorID, 0), //ID = 0 is reserved for RecoveryQuery
		ElevatorID: config.ElevatorID,
	}

	tracker := message.NewAckTracker(recoveryMsg.MsgID, []int{config.ElevatorID})
	e.ch_ackTracker <- tracker

	fmt.Println("[INIT] Checking if backup exists on the network")

	//Query the network to recover information if elevator previously was connected to the network
Loop:
	for {
		select {
		case <-resendTicker.C:
			e.ch_msgTx <- recoveryMsg

		case <-tracker.GetDoneChan():
			break Loop

		case <-timeout:
			fmt.Println("[INIT] Timeout, starting from scratch")
			drivers.SetMotorDirection(drivers.MD_Up)
			foundFloorChan := make(chan int)
			go func() {
				ticker := time.NewTicker(10 * time.Millisecond)
				defer ticker.Stop()

				for {
					<-ticker.C
					currentFloor := drivers.GetFloor()
					if currentFloor != -1 {
						foundFloorChan <- currentFloor
						drivers.SetMotorDirection(drivers.MD_Stop)

						break
					}
				}
			}()

			validFloor := <-foundFloorChan

			e.state = Init
			e.currentFloor = validFloor
			e.requestMatrix = requestmatrix.NewRequestMatrix(config.NumFloors)
			break Loop
		}
	}
	for i := 0; i < config.NumFloors-1; i++ {
		drivers.SetButtonLamp(drivers.BT_Cab, i, e.requestMatrix.GetCabRequest()[i])
	}
}

// Recovers elevator to previous state
func (e *Elevator) RecoverState(stateData *message.ElevatorState) {
	e.requestMatrix.SetAllCabRequest(stateData.RequestMatrix.CabRequests)
	e.travelDirection = Direction(stateData.Direction)

	fmt.Printf("[ElevatorFSM] Recovered cab orders: %v\n", e.requestMatrix.GetCabRequest())

	e.errorTrigger = ErrorType(stateData.ErrorTrigger)
	if stateData.ErrorTrigger != int(ErrorNone) {
		fmt.Printf("[ElevatorFSM] errortype is: %d\n", int(e.errorTrigger))
	}

	e.transitionTo(ElevatorState(stateData.State))
}

func (e *Elevator) chooseDirection() Direction {
	switch e.travelDirection {
	case Up:
		if e.ordersAbove() {
			return Up

		} else if e.anyOrdersAtCurrentFloor() {
			return Stop

		} else if e.ordersBelow() {
			return Down

		} else {
			return Stop
		}

	case Down, Stop:
		if e.ordersBelow() {
			return Down

		} else if e.anyOrdersAtCurrentFloor() {
			return Stop

		} else if e.ordersAbove() {
			return Up

		} else {
			return Stop
		}

	default:
		return Stop
	}
}

func (e *Elevator) GetRequestMatrix() *requestmatrix.RequestMatrix {
	return e.requestMatrix
}

func (e *Elevator) GetStatus() systemdata.ElevatorStatus {
	//var reqMatrix requestmatrix.RequestMatrix
	available := true
	if e.state == Error {
		available = false
	}
	/*
		if e.requestMatrix != nil {
			reqMatrix = *e.requestMatrix
		}
	*/
	return systemdata.ElevatorStatus{
		ElevatorID:      e.ElevatorID,
		State:           int(e.state),
		CurrentFloor:    e.currentFloor,
		TravelDirection: int(e.travelDirection),
		LastUpdated:     time.Now(),
		RequestMatrix:   e.requestMatrix.ToDTO(),
		Available:       available,
		ErrorTrigger:    int(e.errorTrigger),
	}
}

func (e *Elevator) SetHallLigths(matrix [][2]bool) {
	if len(matrix) == 0 {
		fmt.Println("Error: matrix is empty!")
		return
	}

	for i := 0; i < config.NumFloors-1; i++ {
		drivers.SetButtonLamp(drivers.BT_HallUp, i, matrix[i][0])
	}

	for i := 1; i < config.NumFloors; i++ {
		drivers.SetButtonLamp(drivers.BT_HallDown, i, matrix[i][1])
	}
}
