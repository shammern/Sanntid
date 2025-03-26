package elevator

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"fmt"
	"time"
)

func (e *Elevator) Run() {
	for {
		select {
		case order := <-e.ch_orders:
			e.handleNewOrder(order)

		case ev := <-e.ch_fsmEvents:
			e.handleFSMEvent(ev)

		// Case to handle when doortimer elapses to initiad door closing
		case <-func() <-chan time.Time {
			if e.doorTimer == nil {
				return make(chan time.Time)
			}
			return e.doorTimer.C
		}():
			e.ch_fsmEvents <- EventDoorTimerElapsed

		default:
			//Check if elevator should change movement direction
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
			if (e.state == DoorOpen) && time.Since(e.doorOpenStartTime) > config.DoorOpenThreshold {
				e.errorTrigger = ErrorDoorTimeout
				e.ch_fsmEvents <- EventSetError
			}

			// Error check for motor/movement:
			if (e.state == MovingUp || e.state == MovingDown) &&
				drivers.GetFloor() == -1 &&
				time.Since(e.moveStartTime) > config.TimeBetweenFloorsThreshold {
				e.errorTrigger = ErrorMotorFailure
				e.ch_fsmEvents <- EventSetError
			}

			// Attempts recovery if motorerror is present
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

// Takes in a Order consisting of buttonEvent and a flag and updates the requestMatrix
func (e *Elevator) handleNewOrder(newOrder Order) {

	switch newOrder.Event.Button {
	case drivers.BT_Cab:
		e.requestMatrix.SetCabRequest(newOrder.Event.Floor, newOrder.Flag)

	case drivers.BT_HallUp:
		e.requestMatrix.SetHallRequest(newOrder.Event.Floor, 0, newOrder.Flag) 

	case drivers.BT_HallDown:
		e.requestMatrix.SetHallRequest(newOrder.Event.Floor, 1, newOrder.Flag) 
	}

	//Handles new orders on same floor immediately
	if newOrder.Flag {
		if newOrder.Event.Floor == e.currentFloor && e.state == Idle ||
			newOrder.Event.Floor == e.currentFloor && e.state == DoorOpen {
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
		fmt.Println("[ElevatorFSM] Door obstructed")
		e.doorObstructed = true
		//If doortimer is active e.g door is open, the timer will be stopped to keep door from closing
		if e.doorTimer != nil {
			if !e.doorTimer.Stop() {
				select {
				case <-e.doorTimer.C:
				default:
				}
			}
			e.doorTimer = nil
		}

	case EventDoorReleased:
		fmt.Println("[ElevatorFSM] Door releases")
		e.doorObstructed = false
		if e.state == DoorOpen || (e.state == Error && e.errorTrigger == ErrorDoorTimeout) {
			e.transitionTo(DoorOpen)
			e.errorTrigger = ErrorNone
		}

	case EventSetError:
		e.transitionTo(Error)
		drivers.SetMotorDirection(drivers.MD_Stop)
	}
}

// Transitions the statemachine into a new state
func (e *Elevator) transitionTo(newState ElevatorState) {
	e.state = newState
	switch newState {
	case Idle:
		e.travelDirection = Stop
		fmt.Println("[ElevatorFSM] State = Idle")

	case DoorOpen:
		fmt.Println("[ElevatorFSM] State = DoorOpen")
		e.doorOpenStartTime = time.Now()
		if !e.doorObstructed {
			e.doorTimer = time.NewTimer(3 * time.Second)
		}

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
