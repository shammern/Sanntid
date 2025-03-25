package app

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/message"
	"elevator-project/pkg/state"
	"elevator-project/pkg/utils"
	"fmt"
	"time"
)

const masterTimeout = 3 * time.Second

// Monitor master heartbeat and elect a new master if necessary
func MonitorMasterHeartbeat(store *state.Store, msgTx chan message.Message) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		alive := utils.GetAliveElevators(state.MasterStateStore, masterTimeout)

		if contains(alive, CurrentMasterID) {
			continue // Master is still alive
		}

		newMaster, ok := utils.ElectMaster(alive)
		if !ok || newMaster == CurrentMasterID {
			continue
		}

		CurrentMasterID = newMaster
		config.IsMaster = (CurrentMasterID == config.ElevatorID)

		fmt.Printf("[INFO] New master elected: Elevator %d\n", CurrentMasterID)
		go BroadcastMasterAnnouncement(msgTx, CurrentMasterID)
	}
}

func BroadcastMasterAnnouncement(msgTx chan message.Message, masterID int) {
	announcement := message.Message{
		Type:     message.MasterAnnouncement,
		MasterID: masterID,
	}

	ticker := time.NewTicker(config.ResendInterval)
	defer ticker.Stop()
	timeout := time.After(1 * time.Second)

	for {
		select {
		case <-ticker.C:
			msgTx <- announcement
		case <-timeout:
			return
		}
	}
}

// Util func for checking if a value exists in a slice
func contains(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
