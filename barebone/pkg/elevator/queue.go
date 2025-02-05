package elevator

import (
	"barebone/pkg/drivers"
	"fmt"
)

type Order struct {
	ID    int
	Floor int
}

func QueueManager(newOrders <-chan drivers.ButtonEvent, elevatorReady <-chan bool, sendOrder chan<- drivers.ButtonEvent) {
	var queue []drivers.ButtonEvent

	for {
		select {
		// Mottar ny bestilling
		case o := <-newOrders:
			fmt.Printf("[QueueManager] Ny bestilling lagt i køen: %#v\n", o)
			queue = append(queue, o)

		// Heisen signaliserer at den er klar
		case <-elevatorReady:
			if len(queue) > 0 {
				next := queue[0]
				queue = queue[1:]
				fmt.Printf("[QueueManager] Sender bestilling til heisen: %#v\n", next)
				sendOrder <- next
			} else {
				fmt.Println("[QueueManager] Ingen ordre i kø.")
			}
		}
	}
}
