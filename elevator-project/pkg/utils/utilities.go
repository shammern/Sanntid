package utils

import (
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/message"
	"elevator-project/pkg/network/peers"
	"strconv"
)

func ButtonTypeToString(b drivers.ButtonType) string {
	switch b {
	case drivers.BT_HallUp:
		return "Hallcall up"
	case drivers.BT_HallDown:
		return "Hallcall down"
	case drivers.BT_Cab:
		return "Cab call"
	default:
		return "Unknown button"
	}
}

func MessageTypeToString(m message.MessageType) string {
	switch m {
	case message.ElevatorStatus:
		return "State"
	case message.ButtonEvent:
		return "ButtonEvent"
	case message.OrderDelegation:
		return "OrderDelegation"
	case message.CompletedOrder:
		return "CompletedOrder"
	case message.Ack:
		return "Ack"
	case message.Heartbeat:
		return "Heartbeat"
	case message.MasterQuery:
		return "MasterQuery"
	case message.MasterAnnouncement:
		return "MasterAnnouncement"
	default:
		return "Unknown"
	}
}

func GetActiveElevators() []int {
	activeElevators := peers.LatestPeerUpdate.Peers
	var peerIDs []int
	for _, peerStr := range activeElevators {
		id, _ := strconv.Atoi(peerStr)
		peerIDs = append(peerIDs, id)
	}
	return peerIDs
}

func CompareMaps(m1, m2 map[string][][2]bool) bool {
	for key, slice1 := range m1 {
		slice2, ok := m2[key]
		if !ok {
			return false
		}
		if len(slice1) != len(slice2) {
			return false
		}
		for i, arr1 := range slice1 {
			arr2 := slice2[i]
			if arr1[0] != arr2[0] || arr1[1] != arr2[1] {
				return false
			}
		}
	}
	return true
}
