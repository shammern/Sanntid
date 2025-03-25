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

	startupDelay := time.NewTimer(2 * time.Second)
	<-startupDelay.C

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {

		statuses := state.MasterStateStore.GetAll()
		masterStatus, exists := statuses[CurrentMasterID]
		masterAlive := exists && time.Since(masterStatus.LastUpdated) <= masterTimeout
		isSelfMaster := (CurrentMasterID == config.ElevatorID)

		//log.Printf("[DEBUG] CurrentMasterID: %d, masterAlive: %v\n", CurrentMasterID, config.IsMaster)
		// Update our own master flag
		config.IsMaster = isSelfMaster && masterAlive

		if masterAlive {
			continue // Master is alive, no need to elect a new one
		}

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
