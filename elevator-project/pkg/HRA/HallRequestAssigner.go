package HRA

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/elevator"
	"elevator-project/pkg/message"
	"elevator-project/pkg/state"
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

type HRAInput struct {
	HallRequests [][2]bool               `json:"hallRequests"`
	States       map[string]HRAElevState `json:"states"`
}

type OrderData struct {
	Orders map[string][][2]bool
}

func HRARun(st *state.Store) (map[string][][2]bool, HRAInput, error) {
	hraExecutable := ""
	switch runtime.GOOS {
	case "linux":
		hraExecutable = "hall_request_assigner"
	case "windows":
		hraExecutable = "hall_request_assigner.exe"
	default:
		panic("OS not supported")
	}

	allElevators := st.GetAll()

	statesMap := make(map[string]HRAElevState)
	for id, elev := range allElevators {
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

	input := HRAInput{
		HallRequests: st.HallRequests,
		States:       statesMap,
	}

	//PrintHRAInput(input)
	jsonBytes, err := json.Marshal(input)
	if err != nil {
		return nil, input, fmt.Errorf("json.Marshal error: %v", err)
	}

	ret, err := exec.Command("../"+hraExecutable, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		return nil, input, fmt.Errorf("exec.Command error: %v, output: %s", err, ret)
	}

	output := make(map[string][][2]bool)
	err = json.Unmarshal(ret, &output)
	if err != nil {
		return nil, input, fmt.Errorf("json.Unmarshal error: %v", err)
	}

	// Optionally, print the output

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

func PrintHRAInput(input HRAInput) {
	fmt.Println("HRA Input:")

	fmt.Println("Hall Requests:")
	for floor, req := range input.HallRequests {
		// Each req is an array of two booleans: [Up, Down]
		fmt.Printf("  Floor %d: Up: %t, Down: %t\n", floor, req[0], req[1])
	}

	fmt.Println("States:")
	for id, state := range input.States {
		fmt.Printf("  Elevator %s:\n", id)
		fmt.Printf("    Behavior   : %s\n", state.Behavior)
		fmt.Printf("    Floor      : %d\n", state.Floor)
		fmt.Printf("    Direction  : %s\n", state.Direction)
		fmt.Printf("    CabRequests: %v\n", state.CabRequests)
	}
}

func HRALoop(elevatorFSM *elevator.Elevator, msgTx chan message.Message, trackerChan chan *message.AckTracker, msgID *message.MsgID, orderRequestCh chan<- OrderData) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		if config.IsMaster {
			newOrder, input, err := HRARun(state.MasterStateStore)
			if err != nil {
				fmt.Println("HRA error:", err)
				continue
			}
			if !utils.CompareMaps(newOrder, state.MasterStateStore.CurrentOrders) {
				PrintHRAInput(input)
				fmt.Printf("[HRA] Master sending the output in MsgID: %s\n", message.MsgCounter.Get())
				for k, v := range newOrder {
					fmt.Printf("%6v : %+v\n", k, v)
				}
				state.MasterStateStore.CurrentOrders = newOrder
				elevatorFSM.SetHallLigths(state.MasterStateStore.HallRequests)
				// Send new order to the order sender worker.
				orderRequestCh <- OrderData{Orders: newOrder}

				// Process new order events.
				events := elevator.ConvertOrderDataToOrders(newOrder)
				for _, event := range events {
					elevatorFSM.Orders <- event
				}
			}
		}
	}
}
