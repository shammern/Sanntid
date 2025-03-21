package app

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/message"
	"elevator-project/pkg/state"
	"fmt"
	"time"
)

// Handle master/slave configuration messages
func HandleMasterSlaveMessage(msg message.Message) {
	fmt.Printf("Received master config update: new master is elevator %d\n", msg.ElevatorID)
	CurrentMasterID = msg.ElevatorID
	IsMaster = (config.ElevatorID == msg.ElevatorID)
}

// Monitor master heartbeat and elect a new master if necessary
func MonitorMasterHeartbeat(peerAddrs []string) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		statuses := MasterStateStore.GetAll()
		masterStatus, exists := statuses[CurrentMasterID]

		if !exists || time.Since(masterStatus.LastUpdated) > 5*time.Second {
			candidate := config.ElevatorID
			for id, status := range statuses {
				if id != CurrentMasterID && time.Since(status.LastUpdated) <= 5*time.Second && id < candidate {
					candidate = id
				}
			}
			if config.ElevatorID == candidate {
				PromoteToMaster(peerAddrs)
				break
			}
		}
	}
}

// Promote this elevator to master
func PromoteToMaster(peerAddrs []string) {
	IsMaster = true
	/*CurrentMasterID = LocalElevatorID
	fmt.Printf("Elevator %d is now promoted to master.\n", LocalElevatorID)

	configMsg := message.Message{
		Type:       message.MasterSlaveConfig,
		ElevatorID: LocalElevatorID,
		Seq:        0,
	}

	for _, addr := range peerAddrs {
		if err := transport.SendMessage(configMsg, addr); err != nil {
			fmt.Printf("Error broadcasting master config to %s: %v\n", addr, err)
		}
	}
	*/
}

// Monitor elevator heartbeats and reassign orders if necessary
func MonitorElevatorHeartbeats() {
	if !IsMaster {
		return
	}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		statuses := MasterStateStore.GetAll()
		for id, status := range statuses {
			if id == config.ElevatorID {
				continue
			}
			if time.Since(status.LastUpdated) > 5*time.Second {
				fmt.Printf("Elevator %d heartbeat stale. Reassigning its orders.\n", id)
				ReassignOrders(status)
			}
		}
	}
}

// Reassign orders from a failed elevator to active elevators
func ReassignOrders(failedStatus state.ElevatorStatus) {
	for floor, hallRequests := range failedStatus.RequestMatrix.HallRequests {
		for dir, active := range hallRequests {
			if active {
				fmt.Printf("Reassigning hall request at floor %d, direction %d from failed elevator %d.\n", floor, dir, failedStatus.ElevatorID)
			}
		}
	}
}
