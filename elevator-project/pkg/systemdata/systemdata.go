package systemdata

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/requestmatrix"
	"sync"
	"time"
)

var MasterStateStore = NewStore()

type ElevatorStatus struct {
	ElevatorID      int
	State           int
	Available       bool
	CurrentFloor    int
	TravelDirection int
	LastUpdated     time.Time
	RequestMatrix   requestmatrix.RequestMatrixDTO
	ErrorTrigger    int
}

type SystemData struct {
	mu            sync.RWMutex
	Elevators     map[int]ElevatorStatus
	HallRequests  [][2]bool
	CurrentOrders map[string][][2]bool
}

func NewStore() *SystemData {
	store := &SystemData{
		Elevators:     make(map[int]ElevatorStatus),
		HallRequests:  make([][2]bool, config.NumFloors),
		CurrentOrders: make(map[string][][2]bool),
	}
	return store
}

func (s *SystemData) UpdateElevatorStatus(status ElevatorStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Elevators[status.ElevatorID] = status
}

func (s *SystemData) GetAll() map[int]ElevatorStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	copy := make(map[int]ElevatorStatus)
	for id, status := range s.Elevators {
		copy[id] = status
	}
	return copy
}

func (s *SystemData) SetHallRequest(button drivers.ButtonEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if button.Button != drivers.BT_Cab {
		if button.Floor < 0 || button.Floor >= len(s.HallRequests) {
			return
		}
		s.HallRequests[button.Floor][int(button.Button)] = true
	}
}

func (s *SystemData) ClearHallRequest(button drivers.ButtonEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if button.Floor < 0 || button.Floor >= len(s.HallRequests) {
		return
	}

	if button.Button == drivers.BT_Cab {
		return
	}

	s.HallRequests[button.Floor][int(button.Button)] = false
}

func (s *SystemData) ClearOrderFromElevator(button drivers.ButtonEvent, elevatorID int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if button.Floor < 0 || button.Floor >= len(s.HallRequests) {
		return
	}
	switch button.Button {
	case drivers.BT_Cab:
		s.Elevators[elevatorID].RequestMatrix.CabRequests[button.Floor] = false

	case drivers.BT_HallUp, drivers.MD_Down:
		s.Elevators[elevatorID].RequestMatrix.HallRequests[button.Floor][int(button.Button)] = false
		s.HallRequests[button.Floor][int(button.Button)] = false
	}
}

func (s *SystemData) GetHallRequests() [][2]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.HallRequests
}

func (s *SystemData) SetHallRequests(matrix [][2]bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.HallRequests = matrix
}

// Markes elevators as unavailable to service new calls
func (s *SystemData) UpdateElevatorAvailability(elevatorID int, newState bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := s.Elevators[elevatorID]
	status.Available = newState
	s.Elevators[elevatorID] = status
}
