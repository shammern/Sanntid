package app

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/elevator"
	"strconv"
)

//Takes a ordermap and extract the orders for the currentelevator. Orders are packed into a list and returned. 
func convertOrderDataToOrders(orderData map[string][][2]bool) []elevator.Order {
	var ordersList []elevator.Order
	orders := orderData[strconv.Itoa(config.ElevatorID)]

	for floor, calls := range orders {
		// Hall up call
		if calls[0] { 
			ordersList = append(ordersList, elevator.Order{
				Event: drivers.ButtonEvent{Floor: floor, Button: drivers.BT_HallUp},
				Flag:  true, 
			})
		} else {
			ordersList = append(ordersList, elevator.Order{
				Event: drivers.ButtonEvent{Floor: floor, Button: drivers.BT_HallUp},
				Flag:  false, 
			})
		}
		// Hall down call
		if calls[1] { 
			ordersList = append(ordersList, elevator.Order{
				Event: drivers.ButtonEvent{Floor: floor, Button: drivers.BT_HallDown},
				Flag:  true, 
			})
		} else {
			ordersList = append(ordersList, elevator.Order{
			Event: drivers.ButtonEvent{Floor: floor, Button: drivers.BT_HallDown},
			Flag:  false, 
		})
	}
	}
	
	return ordersList
}