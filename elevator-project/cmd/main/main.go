package main

import (
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/elevator"
	"fmt"
)


func main(){


    numFloors := 4
	

    drivers.Init("localhost:15657", numFloors)
    
    //var d drivers.MotorDirection = drivers.MD_Up
    //drivers.SetMotorDirection(d)
    
    drv_buttons := make(chan drivers.ButtonEvent)
    drv_floors  := make(chan int)
    drv_obstr   := make(chan bool)
    drv_stop    := make(chan bool) 

	queue 		:= elevator.NewOrderQueue()
	fsmInstance := elevator.NewElevatorFSM(queue)   
    
    /*
	go func() {
		drivers.PollButtons(drv_buttons)
        drivers.PollFloorSensor(drv_floors)
        drivers.PollObstructionSwitch(drv_obstr)
        drivers.PollStopButton(drv_stop)
	}()

	*/
	go drivers.PollButtons(drv_buttons)
	go drivers.PollFloorSensor(drv_floors)
    go drivers.PollObstructionSwitch(drv_obstr)
    go drivers.PollStopButton(drv_stop)

	/*
	go func() {
		for order := range queue.NextOrder {
			fmt.Println("📨 Main: Sending order to FSM -> Floor", order)
	
			select {
			case fsmInstance.NextOrder <- order:
				fmt.Println("✅ Order successfully sent to FSM")
			default:
				fmt.Println("🚨 ERROR: FSM is NOT reading `NextOrder`, order lost!")
			}
		}
	}()
	*/
	
    
		for {
			select {
			case buttonEvent := <-drv_buttons:
				fmt.Printf("Button Pressed: %+v\n", buttonEvent)
				drivers.SetButtonLamp(buttonEvent.Button, buttonEvent.Floor, true)
				queue.Enqueue(buttonEvent)
	
			case floor := <-drv_floors:
				fmt.Printf("Arrived at floor: %+v\n", floor)
				fsmInstance.HandleEvent("floor_arrival", floor)
	
			case obstruction := <-drv_obstr:
				fmt.Printf("Obstruction detected: %+v\n", obstruction)
				if obstruction {
					fsmInstance.HandleEvent("obstruction", 1)
				} else {
					fsmInstance.HandleEvent("obstruction", 0)
				}
	
			case stop := <-drv_stop:
				fmt.Printf("Stop button pressed: %+v\n", stop)
				for f := 0; f < numFloors; f++ {
					for b := drivers.ButtonType(0); b < 3; b++ {
						drivers.SetButtonLamp(b, f, false)
					}
				}
			}
		}
	}