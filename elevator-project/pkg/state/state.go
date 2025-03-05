package state

import (
	"elevator-project/pkg/orders"
	"sync"
	"time"
)

type ElevatorStatus struct {
	ElevatorID    int
	State         int
	Direction     int
	CurrentFloor  int
	TargetFloor   int
	LastUpdated   time.Time
	RequestMatrix orders.RequestMatrix
}

type Store struct {
	mu        sync.RWMutex
	elevators map[int]ElevatorStatus
}

func NewStore() *Store {
	return &Store{
		elevators: make(map[int]ElevatorStatus),
	}
}

func (s *Store) UpdateStatus(status ElevatorStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.elevators[status.ElevatorID] = status
}

func (s *Store) UpdateHeartbeat(elevID int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := s.elevators[elevID]
	status.LastUpdated = time.Now()
	s.elevators[elevID] = status
}

func (s *Store) GetAll() map[int]ElevatorStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	copy := make(map[int]ElevatorStatus)
	for id, status := range s.elevators {
		copy[id] = status
	}
	return copy
}
