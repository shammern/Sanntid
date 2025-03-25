package app

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/message"
	"elevator-project/pkg/state"
	"elevator-project/pkg/utils"
	"fmt"
	"time"
)

// Monitor master heartbeat and elect a new master if necessary
func MonitorMasterHeartbeat(store *state.Store, msgTx chan message.Message) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		alive := utils.GetAliveElevators(state.MasterStateStore, config.MasterTimeout)


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


func InitMasterDiscovery(msgTx chan message.Message, masterAnnounced <-chan struct{}) {
	fmt.Println("[INIT] Sending MasterQuery")
	ticker := time.NewTicker(config.ResendInterval)
	defer ticker.Stop()

	MasterQueryTimer := time.NewTimer(config.QueryMasterTimer)
	defer MasterQueryTimer.Stop()

	for {
		select {
		case <-ticker.C:
			msgTx <- message.Message{
				Type:       message.MasterQuery,
				ElevatorID: config.ElevatorID,
			}
		case <-masterAnnounced:
			fmt.Println("[INIT] MasterAnnouncement received")
			return

		case <-MasterQueryTimer.C:
			fmt.Println("[INIT] No MasterAnnouncement received, starting heartbeat/election loop")
			go MonitorMasterHeartbeat(state.MasterStateStore, msgTx)
			return
		}
	}
}

