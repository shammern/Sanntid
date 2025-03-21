package utils

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
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
