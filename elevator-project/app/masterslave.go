package app

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/message"
	"elevator-project/pkg/state"
	"fmt"
	"sort"
	"time"
)

const masterTimeout = 3 * time.Second

// Updates master and backup based on last heartbeats
func RecalculateRoles(store *state.Store) {
	statuses := store.GetAll()
	activeElevators := []int{}
	for id, status := range statuses {
		if time.Since(status.LastUpdated) <= masterTimeout {
			activeElevators = append(activeElevators, id)
		}
	}
	sort.Ints(activeElevators)
	if len(activeElevators) > 0 {
		CurrentMasterID = activeElevators[0]
	} else {
		CurrentMasterID = -1
	}
	if len(activeElevators) > 1 {
		BackupElevatorID = activeElevators[1]
	} else {
		BackupElevatorID = -1
	}
}

// Handle master/slave configuration messages
func HandleMasterSlaveMessage(msg message.Message) {
	fmt.Printf("Received master config update: new master is elevator %d\n", msg.ElevatorID)
	CurrentMasterID = msg.ElevatorID
	config.IsMaster = (config.ElevatorID == msg.ElevatorID)
}

// Monitor master heartbeat and elect a new master if necessary
func MonitorMasterHeartbeat(store *state.Store, msgTx chan message.Message) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {

		statuses := state.MasterStateStore.GetAll()

		masterStatus, exists := statuses[CurrentMasterID]
		// Check if current master is online, if yes do not change master
		if CurrentMasterID != -1 && exists && time.Since(masterStatus.LastUpdated) <= masterTimeout {
			continue
		}

		//Reelect master, based on lowest id
		activeElevators := []int{}
		for id, status := range statuses {
			if time.Since(status.LastUpdated) <= masterTimeout {
				activeElevators = append(activeElevators, id)
			}
		}
		sort.Ints(activeElevators)
		if len(activeElevators) > 0 {
			newMaster := activeElevators[0]
			if newMaster != CurrentMasterID {
				CurrentMasterID = newMaster
				fmt.Printf("[INFO] Ny master er heis %d\n", CurrentMasterID)
				msgTx <- message.Message{
					Type:       message.MasterAnnouncement,
					ElevatorID: CurrentMasterID,
				}
			}
		} else {
			CurrentMasterID = -1
		}
	}
}

// Promote this elevator to master

func PromoteToMaster(peerAddrs []string) {
	config.IsMaster = true
	/*CurrentMasterID = LocalElevatorID
	fmt.Printf("Elevator %d is now promoted to master.\n", LocalElevatorID)


func PromoteToMaster(peerAddrs []string, msgTx chan message.Message) {
	IsMaster = true
	CurrentMasterID = config.ElevatorID
	fmt.Printf("[INFO] Heis %d er nå master!\n", config.ElevatorID)
	MasterStateStore.UpdateHeartbeat(CurrentMasterID)

	configMsg := message.Message{
		Type:       message.MasterAnnouncement,
		ElevatorID: config.ElevatorID,
	}
	msgTx <- configMsg
}


// Reassign orders from a failed elevator to active elevators
func ReassignOrders(failedStatus state.ElevatorStatus) {
	for floor, hallRequests := range failedStatus.RequestMatrix.HallRequests {
		for dir, active := range hallRequests {
			if active {
				fmt.Printf("Reassigning hall request at floor %d, direction %d from failed elevator %d.\n", floor, dir, failedStatus.ElevatorID)

// Monitor elevator heartbeats and reassign orders if necessary
func MonitorElevatorHeartbeats(msgTx chan message.Message) {
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

			}
		}
	}
}
