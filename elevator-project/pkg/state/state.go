package state

import (
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/orders"
	"fmt"
	"sync"
	"time"
)

// ElevatorStatus holds information about an elevator.
type ElevatorStatus struct {
	ElevatorID      int
	State           int
	Direction       int // Should be changed to driver.MD?
	CurrentFloor    int
	TravelDirection int
	LastUpdated     time.Time
	RequestMatrix   orders.RequestMatrix
}

// Store holds a map of ElevatorStatus instances.
type Store struct {
	mu           sync.RWMutex
	elevators    map[int]ElevatorStatus
	HallRequests [][2]bool
}

// NewStore creates a new Store.
func NewStore() *Store {
	return &Store{elevators: make(map[int]ElevatorStatus)}
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

// SetHallLight sets the request for a specific floor and button for the current elevator.
func (s *Store) SetHallRequest(button drivers.ButtonEvent) error {
	//TODO: Add fault check if trying to set a cab button
	s.mu.Lock()
	defer s.mu.Unlock()

	if button.Floor < 0 || button.Floor >= len(s.HallRequests) {
		return fmt.Errorf("floor index %d out of bounds", button.Floor)
	}

	s.HallRequests[button.Floor][int(button.Button)] = true

	return nil
}

// SetHallLight clears the request for a specific floor and button for the current elevator.
func (s *Store) ClearHallRequest(button drivers.ButtonEvent) error {
	//TODO: Add fault check if trying to set a cab button
	s.mu.Lock()
	defer s.mu.Unlock()

	if button.Floor < 0 || button.Floor >= len(s.HallRequests) {
		return fmt.Errorf("floor index %d out of bounds", button.Floor)
	}

	if button.Button == drivers.BT_Cab {
		return nil
	}

	s.HallRequests[button.Floor][int(button.Button)] = false

	return nil
}

// ClearHallLight clears the light for a specific floor and button for all elevators.
func (s *Store) ClearOrder(button drivers.ButtonEvent, elevatorID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if button.Floor < 0 || button.Floor >= len(s.HallRequests) {
		return fmt.Errorf("floor index %d out of bounds", button.Floor)
	}
	switch button.Button {
	case drivers.BT_Cab:
		s.elevators[elevatorID].RequestMatrix.CabRequests[button.Floor] = false

	case drivers.BT_HallUp:
		s.elevators[elevatorID].RequestMatrix.HallRequests[button.Floor][0] = false
		s.HallRequests[button.Floor][int(button.Button)] = false

	case drivers.BT_HallDown:
		s.elevators[elevatorID].RequestMatrix.HallRequests[button.Floor][0] = false
		s.HallRequests[button.Floor][int(button.Button)] = false

	}

	return nil
}

// GetElevatorLights returns the Lights matrix for the given elevator ID.
func (s *Store) GetHallOrders() [][2]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.HallRequests
}

// SetHallLight sets the request for a specific floor and button for the current elevator.
func (s *Store) SetAllHallRequest(matrix [][2]bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.HallRequests = matrix
}
