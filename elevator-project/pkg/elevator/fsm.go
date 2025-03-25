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
	ElevatorID          int
	state               ElevatorState
	currentFloor        int
	travelDirection     Direction
	RequestMatrix       *RM.RequestMatrix
	Orders              chan Order
	fsmEvents           chan FsmEvent
	doorTimer           *time.Timer
	msgTx               chan message.Message
	ackTrackerChan      chan *message.AckTracker
	counter             *message.MsgID
	doorOpenStartTime   time.Time
	moveStartTime       time.Time
	errorTrigger        ErrorType
	lastRecoveryAttempt time.Time
}

func NewElevator(ElevatorID int, msgTx chan message.Message, counter *message.MsgID, trackerChan chan *message.AckTracker) *Elevator {
	return &Elevator{
		ElevatorID:      ElevatorID,
		Orders:          make(chan Order, 10),
		fsmEvents:       make(chan FsmEvent, 10),
		msgTx:           msgTx,
		counter:         counter,
		travelDirection: Stop,
		ackTrackerChan:  trackerChan,
		RequestMatrix:   RM.NewRequestMatrix(config.NumFloors),
	}
}

func (e *Elevator) InitElevator() {
	resendTicker := time.NewTicker(config.ResendInterval)
	defer resendTicker.Stop()

	timeout := time.After(1 * time.Second)

	recoveryMsg := message.Message{
		Type:       message.RecoveryQuery,
		MsgID:      fmt.Sprintf("%d-%d", config.ElevatorID, 0),
		ElevatorID: config.ElevatorID,
	}

	tracker := message.NewAckTracker(recoveryMsg.MsgID, []int{config.ElevatorID})

	e.ackTrackerChan <- tracker

	fmt.Println("[INIT] Checking if backup exists on the network")

	for {
		select {
		case <-resendTicker.C:
			e.msgTx <- recoveryMsg

		case <-tracker.Done:
			return

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
						return
					}
				}
			}()

			validFloor := <-foundFloorChan

			e.state = Init
			e.currentFloor = validFloor
			e.RequestMatrix = RM.NewRequestMatrix(config.NumFloors)
			return
		}
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
			// Error check for door open states:
			if (e.state == DoorOpen || e.state == DoorObstructed) &&
				time.Since(e.doorOpenStartTime) > config.DoorOpenThreshold {
				e.errorTrigger = ErrorDoorTimeout
				e.fsmEvents <- EventSetError
			}

			// Error check for moving states:
			if (e.state == MovingUp || e.state == MovingDown) &&
				drivers.GetFloor() == -1 &&
				time.Since(e.moveStartTime) > config.TimeBetweenFloorsThreshold {
				e.errorTrigger = ErrorMotorFailure
				e.fsmEvents <- EventSetError
			}

			if e.state == Error && e.errorTrigger == ErrorMotorFailure {
				if time.Since(e.lastRecoveryAttempt) > 3*time.Second {
					fmt.Println("[ElevatorFSM: Recovery] Motor error: attempting recovery")
					if e.travelDirection == Up {
						drivers.SetMotorDirection(drivers.MD_Up)

					} else {
						drivers.SetMotorDirection(drivers.MD_Down)

					}
					e.lastRecoveryAttempt = time.Now()

				}
			}

			time.Sleep(10 * time.Millisecond)

		}
	}
}

func (e *Elevator) handleNewOrder(newOrder Order) {

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
		e.moveStartTime = time.Now()
		e.currentFloor = drivers.GetFloor()
		drivers.SetFloorIndicator(e.currentFloor)
		if e.state == Init {
			drivers.SetMotorDirection(drivers.MD_Stop)
			drivers.SetDoorOpenLamp(false)
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
		if e.state == DoorObstructed || e.state == Error {
			e.transitionTo(DoorOpen)
			e.errorTrigger = ErrorNone
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
		e.doorOpenStartTime = time.Now()

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
		e.moveStartTime = time.Now()

	case MovingDown:
		e.travelDirection = Down
		fmt.Println("[ElevatorFSM] State = MovingDown")
		drivers.SetMotorDirection(drivers.MD_Down)
		e.moveStartTime = time.Now()

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
		State:           int(e.state),
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

func (e *Elevator) RecoverState(stateData *message.ElevatorState) {
	e.RequestMatrix.CabRequests = stateData.RequestMatrix.CabRequests
	//e.travelDirection = Direction(stateData.Direction)
	//e.state = ElevatorState(stateData.State)
	fmt.Printf("[ElevatorFSM] Recovered cab orders: %v\n", e.RequestMatrix.CabRequests)
	e.transitionTo(ElevatorState(stateData.State))
}
