package master

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/message"
	"elevator-project/pkg/systemdata"
	"elevator-project/pkg/utils"
	"fmt"
	"time"
)

// Monitor master heartbeat and elect a new master if necessary
func MonitorMasterHeartbeat(store *systemdata.SystemData, msgTx chan message.Message, ch_ackTracker chan *message.AckTracker) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		statuses := systemdata.MasterStateStore.GetAll()
		masterStatus, exists := statuses[config.CurrentMasterID]
		masterAlive := exists && time.Since(masterStatus.LastUpdated) <= config.MasterTimeout

		if masterAlive {
			continue
		}

		electNewMaster(msgTx, ch_ackTracker)
	}
}

func BroadcastMasterAnnouncement(msgTx chan message.Message, masterID int, ch_ackTracker chan *message.AckTracker) {

	ticker := time.NewTicker(config.ResendInterval)
	defer ticker.Stop()
	timeout := time.After(1 * time.Second)

	announcement := message.Message{
		Type:     message.MasterAnnouncement,
		MsgID:    message.MsgCounter.Next(),
		MasterID: masterID,
	}

	tracker := message.NewAckTracker(announcement.MsgID, utils.GetActiveElevators())
	tracker.SetOwnAck()
	ch_ackTracker <- tracker

	for {
		select {
		case <-ticker.C:
			msgTx <- announcement

		case <-timeout:
			return

		case <-tracker.GetDoneChan():
			fmt.Printf("[MH: Master] Terminating sending of MsgID: %s, stopping order broadcast\n", announcement.MsgID)
			return
		}
	}
}

func InitMasterDiscovery(ch_msgTx chan message.Message, ch_ackTracker chan *message.AckTracker, masterAnnounced <-chan struct{}) {
	fmt.Println("[INIT] Sending MasterQuery")
	ticker := time.NewTicker(config.ResendInterval)
	defer ticker.Stop()

	MasterQueryTimer := time.NewTimer(config.QueryMasterTimer)
	defer MasterQueryTimer.Stop()

	for {
		select {
		case <-ticker.C:
			ch_msgTx <- message.Message{
				Type:       message.MasterQuery,
				ElevatorID: config.ElevatorID,
			}
		case <-masterAnnounced:
			fmt.Println("[INIT] MasterAnnouncement received")
			return

		case <-MasterQueryTimer.C:
			fmt.Println("[INIT] No MasterAnnouncement received, starting heartbeat/election loop")
			electNewMaster(ch_msgTx, ch_ackTracker)
			return
		}
	}
}

// Elects new master and broadcast MasterID on network
func electNewMaster(ch_msgTx chan message.Message, ch_ackTracker chan *message.AckTracker) {
	newMaster := utils.GetActiveElevators()[0]
	config.CurrentMasterID = newMaster
	config.IsMaster = (config.CurrentMasterID == config.ElevatorID)

	fmt.Printf("[INFO] New master elected: Elevator %d\n", config.CurrentMasterID)
	go BroadcastMasterAnnouncement(ch_msgTx, config.CurrentMasterID, ch_ackTracker)
}
