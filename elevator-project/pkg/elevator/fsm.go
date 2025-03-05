package elevator

import (
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/orders"
	"elevator-project/pkg/state"
	"fmt"
	"time"
)

type ElevatorState int

const (
	Idle ElevatorState = iota
	MovingUp
	MovingDown
	DoorOpen
	DoorObstructed
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

type Elevator struct {
	ElevatorID	 	int
	state        	ElevatorState
	currentFloor 	int
	targetFloor  	int
	requestMatrix   *orders.RequestMatrix 
	orders       	chan drivers.ButtonEvent
	fsmEvents    	chan FsmEvent
	doorTimer    	*time.Timer
}

func NewElevator(rm *orders.RequestMatrix, elevatorID int) *Elevator {
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
				return
			}
		}
	}()

	validFloor := <-foundFloorChan

	return &Elevator{
		ElevatorID:   elevatorID,
		state:        Idle,
		currentFloor: validFloor,
		requestMatrix:rm,
		orders:       make(chan drivers.ButtonEvent, 10),
		fsmEvents:    make(chan FsmEvent, 10),
	}
}

func (e *Elevator) Run() {
	for {
		select {
		case order := <-e.orders:
			e.handleNewOrder(order)
		case ev := <-e.fsmEvents:
			e.handleFSMEvent(ev)
		case <-func() <-chan time.Time {
			if e.doorTimer == nil {
				return make(chan time.Time)
			}
			return e.doorTimer.C
		}():
			e.fsmEvents <- EventDoorTimerElapsed
		default:
			if e.state == Idle || e.state == MovingUp || e.state == MovingDown {
				e.checkAndAssignOptimalOrder()
			}
			time.Sleep(10 * time.Millisecond) //blocking -> should find a better solution
		}
	}
}

func (e *Elevator) handleNewOrder(order drivers.ButtonEvent) {
	if e.state != Idle && e.state != MovingUp && e.state != MovingDown {
		e.fsmEvents <- EventSetError
		return
	}
	e.targetFloor = order.Floor
	switch {
	case order.Floor == e.currentFloor:
		e.elevatorAtCorrectFloor()
	case order.Floor > e.currentFloor:
		e.transitionTo(MovingUp)
	case order.Floor < e.currentFloor:
		e.transitionTo(MovingDown)
	}
}

func (e *Elevator) handleFSMEvent(ev FsmEvent) {
	switch ev {
	case EventArrivedAtFloor:
		if e.state == MovingUp || e.state == MovingDown {
			e.currentFloor = drivers.GetFloor()
			drivers.SetFloorIndicator(e.currentFloor)
			if e.currentFloor == e.targetFloor {
				e.elevatorAtCorrectFloor()
			}
		}
	case EventDoorTimerElapsed:
		if e.state == DoorOpen {
			drivers.SetDoorOpenLamp(false)
			e.transitionTo(Idle)
		}
	case EventDoorObstructed:
		if e.state == DoorOpen {
			e.transitionTo(DoorObstructed)
		}
	case EventDoorReleased:
		if e.state == DoorObstructed {
			e.transitionTo(DoorOpen)
		}
	case EventSetError:
		e.transitionTo(Error)
		drivers.SetMotorDirection(drivers.MD_Stop)
	}
}

func (e *Elevator) transitionTo(newState ElevatorState) {
	e.state = newState
	switch newState {
	case Idle:
		fmt.Println("[ElevatorFSM] State = Idle")
	case DoorOpen:
		fmt.Println("[ElevatorFSM] State = DoorOpen")
		e.doorTimer = time.NewTimer(3 * time.Second)
	case DoorObstructed:
		if e.doorTimer != nil {
			if !e.doorTimer.Stop() {
				<-e.doorTimer.C
			}
			e.doorTimer = nil
		}
		fmt.Println("[ElevatorFSM] State = DoorObstructed")
	case MovingUp:
		fmt.Println("[ElevatorFSM] State = MovingUp")
		drivers.SetMotorDirection(drivers.MD_Up)
	case MovingDown:
		fmt.Println("[ElevatorFSM] State = MovingDown")
		drivers.SetMotorDirection(drivers.MD_Down)
	case Error:
		fmt.Println("[ElevatorFSM] State = Error")
		drivers.SetMotorDirection(drivers.MD_Stop)
	}
}

func (e *Elevator) checkAndAssignOptimalOrder() {
	order, found := OptimalAssignment(e.requestMatrix, e.currentFloor, e.currentDirection())
	if found {
		if e.state == Idle {
			fmt.Printf("[ElevatorFSM] Optimal order found and cleared (Idle): Floor %d, Button %v\n", order.Floor, order.Button)
			e.handleNewOrder(order)
		} else if e.state == MovingUp {
			if order.Floor > e.currentFloor && order.Floor < e.targetFloor {
				fmt.Printf("[ElevatorFSM] Updating target from %d to %d (MovingUp)\n", e.targetFloor, order.Floor)
				e.targetFloor = order.Floor
			}
		} else if e.state == MovingDown {
			if order.Floor < e.currentFloor && order.Floor > e.targetFloor {
				fmt.Printf("[ElevatorFSM] Updating target from %d to %d (MovingDown)\n", e.targetFloor, order.Floor)
				e.targetFloor = order.Floor
			}
		}
	}
}

func (e *Elevator) currentDirection() drivers.MotorDirection {
	switch e.state {
	case MovingUp:
		return drivers.MD_Up
	case MovingDown:
		return drivers.MD_Down
	default:
		return drivers.MD_Stop
	}
}

func (e *Elevator) UpdateElevatorState(ev FsmEvent) {
	e.fsmEvents <- ev
}

func (e *Elevator) clearAllLigths() {
	for b := drivers.ButtonType(0); b < 3; b++ {
		drivers.SetButtonLamp(b, e.currentFloor, false)
	}
}

func (e *Elevator) elevatorAtCorrectFloor() {
	if e.requestMatrix.CabRequests[e.currentFloor] {
		_ = e.requestMatrix.ClearCabRequest(e.currentFloor)
	}
	if e.requestMatrix.HallRequests[e.currentFloor][0] {
		_ = e.requestMatrix.ClearHallRequest(e.currentFloor, 0)
	}
	if e.requestMatrix.HallRequests[e.currentFloor][1] {
		_ = e.requestMatrix.ClearHallRequest(e.currentFloor, 1)
	}
	drivers.SetMotorDirection(drivers.MD_Stop)
	drivers.SetDoorOpenLamp(true)
	e.clearAllLigths()
	e.transitionTo(DoorOpen)
}

func (e *Elevator) GetStatus() state.ElevatorStatus {
	var reqMatrix orders.RequestMatrix
	if e.requestMatrix != nil {
		reqMatrix = *e.requestMatrix
	}
	return state.ElevatorStatus{
		ElevatorID:    e.ElevatorID,
		State:         int(e.state), 
		CurrentFloor:  e.currentFloor,
		TargetFloor:   e.targetFloor,
		LastUpdated:   time.Now(), 
		RequestMatrix: reqMatrix,
	}
}