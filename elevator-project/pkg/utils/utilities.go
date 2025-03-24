package utils

import (
	"elevator-project/pkg/config"
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

func ButtonIntToString(button int) string {
	switch button {
	case 0:
		return "Hallcall up"
	case 1:
		return "Hallcall down"
	case 2:
		return "Cab call"
	default:
		return "Unknown button"
	}
}

func MessageTypeToString(m message.MessageType) string {
	switch m {
	case message.State:
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

func GetOtherElevatorAddresses(ElevatorID int) []string {
	others := []string{}
	for id, address := range config.UDPAddresses {
		if id != ElevatorID {
			others = append(others, address)
		}
	}
	return others
}

func ElevatorIntToString(num int) string {
	switch num {
	case 1:
		return "one"
	case 2:
		return "two"
	case 3:
		return "three"
	default:
		return ""
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

/*
func PrintAckTracker(a message.AckTracker) {
	fmt.Println("AckTracker Details:")
	fmt.Printf("  MsgID: %s\n", a.MsgID)
	fmt.Printf("  SentTime: %v\n", a.SentTime)
	fmt.Println("  ExpectedAcks:")
	for id, acked := range a.ExpectedAcks {
		fmt.Printf("    Elevator %d: %t\n", id, acked)
	}
}
*/

// CompareMaps returns true if the two maps are equal.
// Two maps are considered equal if they have the same keys, and for each key,
// the corresponding slice of [2]bool arrays is of the same length and contains identical arrays.
func CompareMaps(m1, m2 map[string][][2]bool) bool {
	// Iterate over all keys and slices in m1.
	for key, slice1 := range m1 {
		slice2, ok := m2[key]
		if !ok {
			// Key from m1 not present in m2.
			return false
		}
		// Check if the slices are of equal length.
		if len(slice1) != len(slice2) {
			return false
		}
		// Compare each [2]bool array in the slices.
		for i, arr1 := range slice1 {
			arr2 := slice2[i]
			if arr1[0] != arr2[0] || arr1[1] != arr2[1] {
				return false
			}
		}
	}

	return true
}
