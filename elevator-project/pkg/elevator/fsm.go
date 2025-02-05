package elevator

import (
	"elevator-project/pkg/drivers"
	"fmt"
	"sync"
	"time"
)

// Elevator states
type State int

const (
	Idle State = iota
	DoorOpen
	Moving
	ErrorState
)

type ElevatorFSM struct {
	CurrentState State
	CurrentFloor int
	Direction    drivers.MotorDirection
	RequestQueue [4][3]bool
	Mutex        sync.Mutex
	Processing   bool
	NextOrder    chan int
	Queue        *OrderQueue
}

func NewElevatorFSM(queue *OrderQueue) *ElevatorFSM {
	fsm := &ElevatorFSM{
		CurrentState: Idle,
		CurrentFloor: drivers.GetFloor(), // Start at actual floor
		Direction:    drivers.MD_Stop,
		Processing:   false,
		NextOrder:    make(chan int, 10),
		Queue:        queue,
	}

	if fsm.CurrentFloor == -1 {
		fmt.Println("Elevator started between floors, moving up to find a floor...")
		fsm.changeState(Moving)
		go fsm.waitForValidFloor()
	}
	go fsm.listenForOrders()
	return fsm
}


func (e *ElevatorFSM) waitForValidFloor() {
	for {
		time.Sleep(100 * time.Millisecond) // Small delay to avoid busy looping
		floor := drivers.GetFloor()
		if floor != -1 {
			e.CurrentFloor = floor
			drivers.SetFloorIndicator(floor)
			drivers.SetMotorDirection(drivers.MD_Stop)
			fmt.Println("Found valid floor:", floor)
			e.Queue.ElevatorIdle <- true
			return
		}
	}
}


// listenForOrders keeps FSM waiting for new orders
func (e *ElevatorFSM) listenForOrders() {
	go func() {
		for {
			select {
			case floor := <-e.NextOrder:
				fmt.Println("🚀 FSM received order: Moving to floor", floor)
				e.HandleEvent("button_press", floor)
			default:
				// Prevent CPU overload by adding a short delay
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}



// HandleEvent processes different events in the FSM
func (e *ElevatorFSM) HandleEvent(event string, param int) {
	fmt.Println("🎯 FSM received event:", event, "with param:", param)

	switch event {
	case "button_press":
		fmt.Println("✅ FSM handling button press for floor:", param)
		e.handleButtonPress(param)
	case "floor_arrival":
		fmt.Println("🏢 FSM handling floor arrival at:", param)
		e.handleFloorArrival(param)
	case "door_timeout":
		e.handleDoorTimeout()
	case "obstruction":
		e.handleObstruction(param)
	case "error":
		e.handleError()
	case "stop":
		fmt.Println("🛑 Stop button pressed! Stopping elevator immediately.")
		drivers.SetMotorDirection(drivers.MD_Stop)
		e.clearAllRequests()
		e.changeState(Idle)
	}
}


func (e *ElevatorFSM) handleButtonPress(floor int) {
	fmt.Println("Button pressed at floor:", floor)
	if e.Processing {
		return
	}
	e.RequestQueue[floor][drivers.BT_Cab] = true
	e.Processing = true

	if e.CurrentState == Idle {
		if floor == e.CurrentFloor {
			e.changeState(DoorOpen)
		} else {
			e.changeState(Moving)
		}
	}
}

func (e *ElevatorFSM) handleFloorArrival(floor int) {
	fmt.Println("Arrived at floor:", floor)
	e.CurrentFloor = floor
	drivers.SetFloorIndicator(floor)
	drivers.SetMotorDirection(drivers.MD_Stop)

	// Clear requests at this floor
	for btn := 0; btn < 3; btn++ {
		if e.RequestQueue[floor][btn] {
			e.RequestQueue[floor][btn] = false
			drivers.SetButtonLamp(drivers.ButtonType(btn), floor, false) // Turn off button light
		}
	}

	e.Processing = false
	e.Queue.ElevatorDone <- true // Signal queue that we finished an order

	// Look for next order
	e.setDirection()
	if e.Direction != drivers.MD_Stop {
		e.changeState(Moving)
	} else {
		fmt.Println("No more requests, going idle.")
		e.changeState(Idle)
	}
}



func (e *ElevatorFSM) handleDoorTimeout() {
	fmt.Println("Door timeout, closing door")
	drivers.SetDoorOpenLamp(false)
	e.changeState(Idle)
}

func (e *ElevatorFSM) handleObstruction(isObstructed int) {
	if isObstructed == 1 {
		e.changeState(ErrorState)
	} else {
		if e.CurrentState == ErrorState {
			e.changeState(Idle)
		}
	}
}

func (e *ElevatorFSM) handleError() {
	fmt.Println("Error detected! Stopping elevator.")
	drivers.SetMotorDirection(drivers.MD_Stop)
	e.changeState(ErrorState)
}

func (e *ElevatorFSM) changeState(newState State) {
	fmt.Println("🔄 Changing state to:", newState, "direction before setting:", e.Direction)
	e.CurrentState = newState

	switch newState {
	case Idle:
		fmt.Println("🛑 FSM is now idle, stopping motor.")
		drivers.SetMotorDirection(drivers.MD_Stop)
		e.Queue.ElevatorIdle <- true
	case Moving:
		e.setDirection() // Calculate correct direction before setting motor
		fmt.Println("🚀 FSM setting motor direction to:", e.Direction)
		drivers.SetMotorDirection(e.Direction)
	case DoorOpen:
		drivers.SetDoorOpenLamp(true)
		time.AfterFunc(3*time.Second, func() {
			e.HandleEvent("door_timeout", 0)
		})
	case ErrorState:
		fmt.Println("❌ FSM entering error state, stopping motor.")
		drivers.SetMotorDirection(drivers.MD_Stop)
	}
}



func (e *ElevatorFSM) clearAllRequests() {
	//e.Mutex.Lock()
	//defer e.Mutex.Unlock()

	fmt.Println("Clearing all requests and turning off button lights...")

	// Clear requests from request queue
	for f := 0; f < len(e.RequestQueue); f++ {
		for b := 0; b < 3; b++ {
			e.RequestQueue[f][b] = false
			drivers.SetButtonLamp(drivers.ButtonType(b), f, false) // Turn off button lights
		}
	}

	// Clear any pending orders in the queue
	for len(e.Queue.NextOrder) > 0 {
		<-e.Queue.NextOrder // Drain channel
	}

	fmt.Println("All requests cleared.")
}



func (e *ElevatorFSM) setDirection() {

	fmt.Println("📢 setDirection() was called. Current floor:", e.CurrentFloor)

	hasRequestsAbove := false
	hasRequestsBelow := false

	// Scan for requests above and below
	for f := e.CurrentFloor + 1; f < len(e.RequestQueue); f++ {
		for btn := 0; btn < 3; btn++ {
			if e.RequestQueue[f][btn] {
				hasRequestsAbove = true
				break
			}
		}
	}
	for f := 0; f < e.CurrentFloor; f++ {
		for btn := 0; btn < 3; btn++ {
			if e.RequestQueue[f][btn] {
				hasRequestsBelow = true
				break
			}
		}
	}

	// Determine direction
	oldDirection := e.Direction
	if hasRequestsAbove {
		e.Direction = drivers.MD_Up
	} else if hasRequestsBelow {
		e.Direction = drivers.MD_Down
	} else {
		e.Direction = drivers.MD_Stop
	}

	fmt.Println("🧭 FSM calculated direction:", e.Direction, "(Previous:", oldDirection, ")")
}


