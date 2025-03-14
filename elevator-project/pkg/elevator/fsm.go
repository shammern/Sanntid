package elevator

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/message"
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

// used for internal elevator logic and handlig
type Elevator struct {
	ElevatorID    int
	state         ElevatorState
	currentFloor  int
	targetFloor   int
	RequestMatrix *orders.RequestMatrix //should change the variable name to requestMatrix
	orders        chan drivers.ButtonEvent
	fsmEvents     chan FsmEvent
	doorTimer     *time.Timer
	msgTx         chan message.Message
	counter       *message.MsgID
}

func NewElevator(ElevatorID int, msgTx chan message.Message, counter *message.MsgID) *Elevator {
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
		ElevatorID:    ElevatorID,
		state:         Idle,
		currentFloor:  validFloor,
		RequestMatrix: orders.NewRequestMatrix(config.NumFloors),
		orders:        make(chan drivers.ButtonEvent, 10),
		fsmEvents:     make(chan FsmEvent, 10),
		msgTx:         msgTx,
		counter:       counter,
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

// checkAndAssignOptimalOrder queries the request matrix for an optimal order.
// If the elevator is idle, it clears the order immediately and processes it.
// If the elevator is moving, it updates the target floor if the new order lies
// between the current floor and the current target, but leaves the request intact.
func (e *Elevator) checkAndAssignOptimalOrder() {
	order, found := OptimalAssignment(e.RequestMatrix, e.currentFloor, e.currentDirection())
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
	completedOrderMsg := message.Message{
		Type:       message.CompletedOrder,
		ElevatorID: config.ElevatorID,
		MsgID:      e.counter.Next(),
	}

	if e.RequestMatrix.CabRequests[e.currentFloor] {
		_ = e.RequestMatrix.ClearCabRequest(e.currentFloor)
	}
	if e.state == MovingUp {
		if e.RequestMatrix.HallRequests[e.currentFloor][0] {
			_ = e.RequestMatrix.ClearHallRequest(e.currentFloor, 0)
			completedOrderMsg.OrderData = drivers.ButtonEvent{Floor: e.currentFloor, Button: drivers.BT_HallUp}
		}
	} else if e.state == MovingDown {
		if e.RequestMatrix.HallRequests[e.currentFloor][1] {
			_ = e.RequestMatrix.ClearHallRequest(e.currentFloor, 1)
			completedOrderMsg.OrderData = drivers.ButtonEvent{Floor: e.currentFloor, Button: drivers.BT_HallDown}
		}
	}

	drivers.SetMotorDirection(drivers.MD_Stop)
	drivers.SetDoorOpenLamp(true)
	e.clearAllLigths()
	e.transitionTo(DoorOpen)

	//include updated RequestMatrix
	//completedOrderMsg.StateData.RequestMatrix = *e.RequestMatrix

	//Notify network that an order has been completed
	e.msgTx <- completedOrderMsg
}

// GetStatus returns a state.ElevatorStatus with the current state of the elevator.
// The LastUpdated field is set to time.Now() at the moment of calling this method.
func (e *Elevator) GetStatus() state.ElevatorStatus {
	// If requestMatrix is stored as a pointer internally, we can dereference it.
	var reqMatrix orders.RequestMatrix
	if e.RequestMatrix != nil {
		reqMatrix = *e.RequestMatrix
	}
	return state.ElevatorStatus{
		ElevatorID:    e.ElevatorID,
		State:         int(e.state), //cant export state, look into this later
		CurrentFloor:  e.currentFloor,
		TargetFloor:   e.targetFloor,
		LastUpdated:   time.Now(), // or use a stored timestamp if you maintain one
		RequestMatrix: reqMatrix,
	}
}

func (e *Elevator) GetRequestMatrix() *orders.RequestMatrix {
	return e.RequestMatrix
}

func (e *Elevator) SetHallLigths(matrix [][]bool) {
	for i := 0; i < config.NumFloors-1; i++ {
		drivers.SetButtonLamp(drivers.BT_HallUp, i, matrix[i][0])
	}
	for i := 1; i < config.NumFloors; i++ {
		drivers.SetButtonLamp(drivers.BT_HallDown, i, matrix[i][1])
	}
}
