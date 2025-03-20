package utils

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/message"
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
	case message.MasterSlaveConfig:
		return "MasterSlaveConfig"
	case message.Promotion:
		return "Promotion"
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
