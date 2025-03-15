package state

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/orders"
	"fmt"
	"sync"
	"time"
)

// ElevatorStatus holds information about an elevator.
type ElevatorStatus struct {
	ElevatorID    int
	State         int
	Direction     int // Should be changed to driver.MD?
	CurrentFloor  int
	TargetFloor   int
	LastUpdated   time.Time
	RequestMatrix orders.RequestMatrix
	Lights        [][]bool
}

// Store holds a map of ElevatorStatus instances.
type Store struct {
	mu        sync.RWMutex
	elevators map[int]ElevatorStatus
}

// NewStore creates a new Store.
func NewStore() *Store {
	return &Store{elevators: make(map[int]ElevatorStatus)}
}

// NewElevatorStatus is a helper to initialize an ElevatorStatus with properly allocated Lights.
func NewElevatorStatus(elevatorID int) ElevatorStatus {
	lights := make([][]bool, config.NumFloors)
	for i := 0; i < config.NumFloors; i++ {
		lights[i] = make([]bool, 2)
	}
	return ElevatorStatus{
		ElevatorID:  elevatorID,
		LastUpdated: time.Now(),
		Lights:      lights,
		// Initialize other fields as needed.
	}
}

func InsertElevators(s *Store) {
	// Define the number of floors and buttons per floor.

	for _, id := range []int{1, 2, 3} {
		status := NewElevatorStatus(id)
		s.UpdateStatus(status)
	}
}

// UpdateStatus updates or adds an ElevatorStatus to the store.
func (s *Store) UpdateStatus(status ElevatorStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.elevators[status.ElevatorID] = status
}

// UpdateHeartbeat updates the heartbeat timestamp for a given elevator.
func (s *Store) UpdateHeartbeat(elevID int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := s.elevators[elevID]
	status.LastUpdated = time.Now()
	s.elevators[elevID] = status
}

// GetAll returns a copy of all elevator statuses.
func (s *Store) GetAll() map[int]ElevatorStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	copy := make(map[int]ElevatorStatus)
	for id, status := range s.elevators {
		copy[id] = status
	}
	return copy
}

// SetHallLight sets the light for a specific floor and button for the current elevator.
func (s *Store) SetHallLight(floor int, button int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := s.elevators[config.ElevatorID]

	// Check floor index.
	if floor < 0 || floor >= len(status.Lights) {
		return fmt.Errorf("floor index %d out of bounds", floor)
	}

	// Check button index.
	if button < 0 || button >= len(status.Lights[floor]) {
		return fmt.Errorf("button index %d out of bounds for floor %d", button, floor)
	}

	// Set the light.
	status.Lights[floor][button] = true
	// Write the updated status back to the store.
	s.elevators[config.ElevatorID] = status
	return nil
}

// ClearHallLight clears the light for a specific floor and button for all elevators.
func (s *Store) ClearHallLight(floor int, button int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, elevator := range s.elevators {
		if floor < len(elevator.Lights) && button < len(elevator.Lights[floor]) {
			elevator.Lights[floor][button] = false
			s.elevators[id] = elevator
		}
	}
}

// GetElevatorLights returns the Lights matrix for the given elevator ID.
func (s *Store) GetElevatorLights(elevatorID int) [][]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := s.elevators[elevatorID]
	return status.Lights
}
