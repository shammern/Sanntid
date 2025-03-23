package elevator

import (
	RM "elevator-project/pkg/RequestMatrix"
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/message"
	"elevator-project/pkg/state"
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

type Direction int

const (
	Up   Direction = 1
	Down           = -1
	Stop           = 0
)

type Order struct {
	Event drivers.ButtonEvent
	Flag  bool
}

// used for internal elevator logic and handlig
type Elevator struct {
	ElevatorID      int
	state           ElevatorState
	currentFloor    int
	travelDirection Direction
	RequestMatrix   *RM.RequestMatrix
	Orders          chan Order
	fsmEvents       chan FsmEvent
	doorTimer       *time.Timer
	msgTx           chan message.Message
	ackTrackerChan  chan *message.AckTracker
	counter         *message.MsgID
}

func NewElevator(ElevatorID int, msgTx chan message.Message, counter *message.MsgID, trackerChan chan *message.AckTracker) *Elevator {
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
		ElevatorID:      ElevatorID,
		state:           Init,
		currentFloor:    validFloor,
		RequestMatrix:   RM.NewRequestMatrix(config.NumFloors),
		Orders:          make(chan Order, 10),
		fsmEvents:       make(chan FsmEvent, 10),
		msgTx:           msgTx,
		counter:         counter,
		travelDirection: Stop,
		ackTrackerChan:  trackerChan,
	}
}

func (e *Elevator) Run() {
	for {
		select {
		case order := <-e.Orders:
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
				newDirection := e.chooseDirection()
				if newDirection != e.travelDirection {
					switch newDirection {
					case Up:
						e.transitionTo(MovingUp)
					case Down:
						e.transitionTo(MovingDown)
					case Stop:
						e.transitionTo(Idle)
					}

					e.travelDirection = newDirection
				}
			}
			time.Sleep(10 * time.Millisecond) //blocking -> should find a better solution
		}
	}
}

// TODO: It might be cleaner if handleNewOrder receives a whole requestMatrix rather than multiple orders.
func (e *Elevator) handleNewOrder(newOrder Order) {
	//fmt.Printf("[ElevatorFSM] New order received type: %d, floor: %d\n", int(order.Button), order.Floor)

	switch newOrder.Event.Button {
	case drivers.BT_Cab:
		e.RequestMatrix.CabRequests[newOrder.Event.Floor] = newOrder.Flag

	case drivers.BT_HallUp:
		e.RequestMatrix.HallRequests[newOrder.Event.Floor][0] = newOrder.Flag

	case drivers.BT_HallDown:
		e.RequestMatrix.HallRequests[newOrder.Event.Floor][1] = newOrder.Flag
	}

	if newOrder.Flag {
		if newOrder.Event.Floor == e.currentFloor && e.state == Idle ||
			newOrder.Event.Floor == e.currentFloor && e.state == DoorOpen ||
			newOrder.Event.Floor == e.currentFloor && e.state == DoorObstructed {
			fmt.Printf("[ElevatorFSM] Received order on same floor. Ordertype: %d, floor: %d\n", int(newOrder.Event.Button), newOrder.Event.Floor)
			//drivers.SetButtonLamp(order.Button, order.Floor, false)
			e.clearHallReqsAtFloor()
			drivers.SetDoorOpenLamp(true)
			e.transitionTo(DoorOpen)
			return
		}
	}
}

func (e *Elevator) handleFSMEvent(ev FsmEvent) {
	switch ev {
	case EventArrivedAtFloor:
		e.currentFloor = drivers.GetFloor()
		drivers.SetFloorIndicator(e.currentFloor)
		if e.state == Init {
			drivers.SetMotorDirection(drivers.MD_Stop)
			drivers.SetDoorOpenLamp(false) // Sikrer at døren forblir lukket
			e.transitionTo(Idle)
			return
		}
		if e.shouldStop() {
			go e.clearHallReqsAtFloor()
			drivers.SetMotorDirection(drivers.MD_Stop)
			drivers.SetDoorOpenLamp(true)
			e.transitionTo(DoorOpen)
		}
	case EventDoorTimerElapsed:
		if e.state == DoorOpen {
			drivers.SetDoorOpenLamp(false)
			newDirection := e.chooseDirection()
			switch newDirection {
			case Stop:
				e.transitionTo(Idle)

			case Up:
				e.transitionTo(MovingUp)

			case Down:
				e.transitionTo(MovingDown)
			}
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
	case Init:
		fmt.Println("[ElevatorFSM] State = Init")
	case Idle:
		e.travelDirection = Stop
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
		e.travelDirection = Up
		fmt.Println("[ElevatorFSM] State = MovingUp")
		drivers.SetMotorDirection(drivers.MD_Up)
	case MovingDown:
		e.travelDirection = Down
		fmt.Println("[ElevatorFSM] State = MovingDown")
		drivers.SetMotorDirection(drivers.MD_Down)
	case Error:
		fmt.Println("[ElevatorFSM] State = Error")
		drivers.SetMotorDirection(drivers.MD_Stop)
	}
}

func (e *Elevator) UpdateElevatorState(ev FsmEvent) {
	e.fsmEvents <- ev
}

func (e *Elevator) GetStatus() state.ElevatorStatus {
	var reqMatrix RM.RequestMatrix
	available := true
	if e.state == Error {
		available = false
	}
	if e.RequestMatrix != nil {
		reqMatrix = *e.RequestMatrix
	}
	return state.ElevatorStatus{
		ElevatorID:      e.ElevatorID,
		State:           int(e.state), //cant export state, look into this later
		CurrentFloor:    e.currentFloor,
		TravelDirection: int(e.travelDirection),
		LastUpdated:     time.Now(),
		RequestMatrix:   reqMatrix,
		Available:       available,
	}
}

func (e *Elevator) GetRequestMatrix() *RM.RequestMatrix {
	return e.RequestMatrix
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

// In package elevator
func (e *Elevator) PrintRequestMatrix() {
	fmt.Println("Request Matrix:")

	// Print Cab Requests
	fmt.Println("Cab Requests:")
	for floor, req := range e.RequestMatrix.CabRequests {
		fmt.Printf("  Floor %d: %v\n", floor, req)
	}

	// Print Hall Requests
	fmt.Println("Hall Requests:")
	for floor, hallReq := range e.RequestMatrix.HallRequests {
		// Each hallReq is an array with two booleans:
		// hallReq[0] for the "up" button and hallReq[1] for the "down" button.
		fmt.Printf("  Floor %d: Up: %v, Down: %v\n", floor, hallReq[0], hallReq[1])
	}
}
