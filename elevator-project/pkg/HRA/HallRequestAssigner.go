package HRA

import (
	"elevator-project/pkg/state"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
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
	case 0:
		return "idle"
	case 1:
		return "moving"
	case 2:
		return "moving"
	case 3:
		return "doorOpen"
	case 4:
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
