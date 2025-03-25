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
				//fmt.Printf("[INFO] Ny master er heis %d\n", CurrentMasterID)
				msgTx <- message.Message{
					Type:     message.MasterAnnouncement,
					MasterID: CurrentMasterID,
				}
				config.IsMaster = (CurrentMasterID == config.ElevatorID)
				fmt.Printf("Masterchange from MonitorMasterHeartbeat: %v\n", config.IsMaster)
			}
		} else {
			CurrentMasterID = -1
		}
	}
}
