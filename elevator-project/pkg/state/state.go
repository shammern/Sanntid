package state

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	RM "elevator-project/pkg/requestmatrix"
	"fmt"
	"strconv"
	"sync"
	"time"
)

var MasterStateStore = NewStore()

// ElevatorStatus holds information about an elevator.
type ElevatorStatus struct {
	ElevatorID      int
	State           int
	Available       bool
	CurrentFloor    int
	TravelDirection int
	LastUpdated     time.Time
	RequestMatrix   RM.RequestMatrixDTO
	ErrorTrigger    int
}

// Store holds a map of ElevatorStatus instances.
type Store struct {
	mu            sync.RWMutex
	Elevators     map[int]ElevatorStatus
	HallRequests  [][2]bool
	CurrentOrders map[string][][2]bool
}

// NewStore creates a new Store.
func NewStore() *Store {
	store := &Store{
		Elevators:     make(map[int]ElevatorStatus),      // start empty
		HallRequests:  make([][2]bool, config.NumFloors), // still need one entry per floor
		CurrentOrders: make(map[string][][2]bool),        // start empty
	}
	return store
}

// UpdateStatus updates or adds an ElevatorStatus to the store.
func (s *Store) UpdateStatus(status ElevatorStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Elevators[status.ElevatorID] = status
	key := strconv.Itoa(status.ElevatorID)
	if _, exists := s.CurrentOrders[key]; !exists {
		s.CurrentOrders[key] = make([][2]bool, config.NumFloors)
	}
}

// GetAll returns a copy of all elevator statuses.
func (s *Store) GetAll() map[int]ElevatorStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	copy := make(map[int]ElevatorStatus)
	for id, status := range s.Elevators {
		copy[id] = status
	}
	return copy
}

// SetHallRequest sets the request for a specific floor and button for the current elevator.
func (s *Store) SetHallRequest(button drivers.ButtonEvent) error {
	//TODO: Add fault check if trying to set a cab button
	s.mu.Lock()
	defer s.mu.Unlock()

	if button.Button != drivers.BT_Cab {
		if button.Floor < 0 || button.Floor >= len(s.HallRequests) {
			return fmt.Errorf("floor index %d out of bounds", button.Floor)
		}

		s.HallRequests[button.Floor][int(button.Button)] = true
	}
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

// ClearHallLight clears the light for a specific floor and button for a specific elevator.
func (s *Store) ClearOrder(button drivers.ButtonEvent, elevatorID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if button.Floor < 0 || button.Floor >= len(s.HallRequests) {
		return fmt.Errorf("floor index %d out of bounds", button.Floor)
	}
	switch button.Button {
	case drivers.BT_Cab:
		s.Elevators[elevatorID].RequestMatrix.CabRequests[button.Floor] = false

	case drivers.BT_HallUp, drivers.MD_Down:
		s.Elevators[elevatorID].RequestMatrix.HallRequests[button.Floor][int(button.Button)] = false
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

func (s *Store) UpdateElevatorAvailability(elevatorID int, newState bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := s.Elevators[elevatorID]
	status.Available = newState
	s.Elevators[elevatorID] = status
}
