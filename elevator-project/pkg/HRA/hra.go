package HRA

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/elevator"
	"elevator-project/pkg/message"
	"elevator-project/pkg/systemdata"
	"elevator-project/pkg/utils"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"time"
)

type HRAElevState struct {
	Behavior    string `json:"behaviour"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests []bool `json:"cabRequests"`
}

type Input struct {
	HallRequests [][2]bool               `json:"hallRequests"`
	States       map[string]HRAElevState `json:"states"`
}

type Output struct {
	Orders map[string][][2]bool
}

// Worker that continuously calculates optimal order assignment to available elevators
func HRAWorker(elevatorFSM *elevator.Elevator, msgTx chan message.Message, trackerChan chan *message.AckTracker, msgID *message.MsgID, orderRequestCh chan<- Output) {
	ticker := time.NewTicker(config.OrderCalculationPeriod)
	defer ticker.Stop()

	for range ticker.C {
		if config.IsMaster {
			newOrder, input, err := calculateOptimalOrderAssignment(systemdata.MasterStateStore)
			if err != nil {
				fmt.Println("HRA error:", err)
				continue
			}

			//Checks wether newly calculated orders are different from the currelty active order
			if !utils.CompareMaps(newOrder, systemdata.MasterStateStore.CurrentOrders) {

				PrintHRAInput(input)
				fmt.Printf("[HRA] Master sending the output in MsgID: %s\n", message.MsgCounter.Get())
				for k, v := range newOrder {
					fmt.Printf("%6v : %+v\n", k, v)
				}
				systemdata.MasterStateStore.CurrentOrders = newOrder
				elevatorFSM.SetHallLigths(systemdata.MasterStateStore.HallRequests)

				// Send new order to the order sender worker to be broadcasted
				orderRequestCh <- Output{Orders: newOrder}

				// Passes own orders to elevatorFSM
				orders := elevator.ConvertOrderDataToOrders(newOrder)
				for _, order := range orders {
					elevatorFSM.SendOrderToFSM(order)
				}
			}
		}
	}
}

// Collects all hallrequests from statestore and returns optimal order assigment
func calculateOptimalOrderAssignment(st *systemdata.SystemData) (map[string][][2]bool, Input, error) {
	hraExecutable := ""
	switch runtime.GOOS {
	case "linux":
		hraExecutable = "hall_request_assigner"
	case "windows":
		hraExecutable = "hall_request_assigner.exe"
	default:
		panic("OS not supported")
	}

	//Extracts relevant informaion from statestores and repackages into an HRA input
	allElevators := st.GetAll()
	statesMap := make(map[string]HRAElevState)
	for id, elev := range allElevators {
		//Checks if elevator is available to take hall orders
		if elev.Available {

			dirString := DirectionIntToString(elev.TravelDirection)
			stateString := StateIntToString(elev.State)

			statesMap[strconv.Itoa(id)] = HRAElevState{
				Behavior:    stateString,
				Floor:       elev.CurrentFloor,
				Direction:   dirString,
				CabRequests: elev.RequestMatrix.CabRequests,
			}
		}
	}

	input := Input{
		HallRequests: st.HallRequests,
		States:       statesMap,
	}

	//PrintHRAInput(input)

	jsonBytes, err := json.Marshal(input)
	if err != nil {
		return nil, input, fmt.Errorf("json.Marshal error: %v", err)
	}

	//Calls external binary file to calculate optimal orders
	ret, err := exec.Command("../"+hraExecutable, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		return nil, input, fmt.Errorf("exec.Command error: %v, output: %s", err, ret)
	}

	output := make(map[string][][2]bool)
	err = json.Unmarshal(ret, &output)
	if err != nil {
		return nil, input, fmt.Errorf("json.Unmarshal error: %v", err)
	}

	return output, input, nil
}

func DirectionIntToString(dir int) string {
	switch dir {
	case 0:
		return "stop"
	case -1:
		return "down"
	case 1:
		return "up"
	default:
		return "Unknown button"
	}
}

func StateIntToString(state int) string {
	switch state {
	case 0, 1:
		return "idle"

	case 2, 3:
		return "moving"

	case 4, 5:
		return "doorOpen"

	default:
		return "Unknown button"
	}
}

func PrintHRAInput(input Input) {
	fmt.Println("HRA Input:")
	fmt.Println("Hall Requests:")
	for floor, req := range input.HallRequests {
		fmt.Printf("  Floor %d: Up: %t, Down: %t\n", floor, req[0], req[1])
	}
	fmt.Println("States:")
	for id, systemdata := range input.States {
		fmt.Printf("  Elevator %s:\n", id)
		fmt.Printf("    Behavior   : %s\n", systemdata.Behavior)
		fmt.Printf("    Floor      : %d\n", systemdata.Floor)
		fmt.Printf("    Direction  : %s\n", systemdata.Direction)
		fmt.Printf("    CabRequests: %v\n", systemdata.CabRequests)
	}
}
