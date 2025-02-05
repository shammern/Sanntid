package elevator

import (
	"fmt"
	"time"
)

type ElevatorState int

const (
	Idle ElevatorState = iota
	MovingUp
	MovingDown
	DoorOpen
	Error
)

type Elevator struct {
	state        ElevatorState
	currentFloor int
	targetFloor  int

	// Kanaler for kommunikasjon
	orders    <-chan Order  // Mottar ordre fra QueueManager
	ready     chan<- bool   // Signaliserer til QueueManager at heisen er klar
	fsmEvents chan fsmEvent // Intern kanal for å trigge state-machine-hendelser
}

type fsmEvent int

const (
	eventArrivedNextFloor fsmEvent = iota
	eventDoorTimerElapsed
	eventSetError
)

// NewElevator er en hjelpefunksjon som lager en Elevator med de tilhørende kanalene
func NewElevator(orders <-chan Order, ready chan<- bool) *Elevator {
	return &Elevator{
		state:        Idle,
		currentFloor: 0,
		orders:       orders,
		ready:        ready,
		fsmEvents:    make(chan fsmEvent),
	}
}

// Run starter heisens state-machine-løkke
func (e *Elevator) Run() {
	// For å simulere bevegelse etasje for etasje,
	// lager vi en ticker som "flytter" heisen hvert sekund
	movementTicker := time.NewTicker(1 * time.Second)
	defer movementTicker.Stop()

	// doorTimer vil bli startet hver gang vi åpner døra (DoorOpen).
	// Vi stopper timere i en `select` når vi er ferdige med dør-åpning.
	var doorTimer *time.Timer

	for {
		select {
		// 1. Lytt på nye ordre (fra QueueManager)
		case order := <-e.orders:
			e.handleNewOrder(order)

		// 2. Lytt på bevegelse for hver "tick", men bare hvis vi er i MovingUp/MovingDown
		case <-movementTicker.C:
			e.handleMovementTick()

		// 3. Lytt på interne FSM-events (f.eks. ankommet neste etasje, dør-timer utløpt)
		case ev := <-e.fsmEvents:
			e.handleFSMEvent(ev)

		// 4. Håndter dør-timeren
		case <-func() <-chan time.Time {
			// For å unngå nil-pointer, sjekker vi om doorTimer er satt
			if doorTimer == nil {
				return make(chan time.Time) // tom kanal som aldri sender
			}
			return doorTimer.C
		}():
			// Door-timer har utløpt -> vi sender eventDoorTimerElapsed
			e.fsmEvents <- eventDoorTimerElapsed
			doorTimer = nil
		}
	}
}

// handleNewOrder tar imot en ny ordre når heisen er i Idle,
// eller setter heisen i Error-tilstand hvis den ikke er klar
func (e *Elevator) handleNewOrder(order Order) {
	fmt.Printf("[ElevatorFSM] Ny ordre mottatt: ID=%d, Floor=%d\n", order.ID, order.Floor)

	// Hvis vi allerede er opptatt (ikke i Idle) kan vi evt. avvise eller behandle senere.
	// Her setter vi bare en slags "feil"-tilstand for å demonstrere:
	if e.state != Idle {
		fmt.Printf("[ElevatorFSM] Kan ikke ta ny ordre: heisen er i tilstand %v\n", e.state)
		e.fsmEvents <- eventSetError
		return
	}

	// Ellers godtar vi ordren
	e.targetFloor = order.Floor
	switch {
	case e.targetFloor == e.currentFloor:
		// Samme etasje -> direkte til DoorOpen
		fmt.Println("[ElevatorFSM] Ordre er i samme etasje -> åpner dør")
		e.transitionTo(DoorOpen)
	case e.targetFloor > e.currentFloor:
		fmt.Println("[ElevatorFSM] Flytter til MovingUp")
		e.transitionTo(MovingUp)
	case e.targetFloor < e.currentFloor:
		fmt.Println("[ElevatorFSM] Flytter til MovingDown")
		e.transitionTo(MovingDown)
	}
}

// handleMovementTick håndterer at heisen beveger seg én etasje per tick
func (e *Elevator) handleMovementTick() {
	switch e.state {
	case MovingUp:
		e.currentFloor++
		fmt.Printf("[ElevatorFSM] Beveger opp -> etasje %d\n", e.currentFloor)
		// Sjekk om vi har ankommet målet
		if e.currentFloor >= e.targetFloor {
			e.fsmEvents <- eventArrivedNextFloor
		}
	case MovingDown:
		e.currentFloor--
		fmt.Printf("[ElevatorFSM] Beveger ned -> etasje %d\n", e.currentFloor)
		// Sjekk om vi har ankommet målet
		if e.currentFloor <= e.targetFloor {
			e.fsmEvents <- eventArrivedNextFloor
		}
	}
	// Hvis heisen er i Idle, DoorOpen eller Error gjør vi ingenting på tick
}

// handleFSMEvent håndterer interne hendelser
func (e *Elevator) handleFSMEvent(ev fsmEvent) {
	switch ev {
	case eventArrivedNextFloor:
		// Har ankommet målet
		if e.state == MovingUp || e.state == MovingDown {
			fmt.Printf("[ElevatorFSM] Ankommet måletasje (%d). Åpner dør.\n", e.currentFloor)
			e.transitionTo(DoorOpen)
		}
	case eventDoorTimerElapsed:
		// Døren er ferdig å være åpen -> Lukk dør og gå til Idle
		if e.state == DoorOpen {
			fmt.Printf("[ElevatorFSM] Dør er lukket. Går til Idle.\n")
			e.transitionTo(Idle)
		}
	case eventSetError:
		e.transitionTo(Error)
	}
}

// transitionTo utfører overgang til en ny tilstand og utfører evt. logikk
func (e *Elevator) transitionTo(newState ElevatorState) {
	e.state = newState
	switch newState {
	case Idle:
		// Signaler at heisen er klar til å motta ny ordre
		fmt.Println("[ElevatorFSM] Tilstand = Idle -> Heisen er ledig.")
		e.ready <- true
	case DoorOpen:
		// Åpne dør: start en timer på f.eks. 2 sekunder før vi lukker
		fmt.Printf("[ElevatorFSM] Tilstand = DoorOpen -> Åpner døra i etasje %d.\n", e.currentFloor)
		go func() {
			time.Sleep(2 * time.Second)
			e.fsmEvents <- eventDoorTimerElapsed
		}()
	case MovingUp:
		fmt.Println("[ElevatorFSM] Tilstand = MovingUp -> Starter bevegelse oppover.")
	case MovingDown:
		fmt.Println("[ElevatorFSM] Tilstand = MovingDown -> Starter bevegelse nedover.")
	case Error:
		fmt.Println("[ElevatorFSM] Tilstand = Error -> Heisen er i feilmodus.")
	}
}
