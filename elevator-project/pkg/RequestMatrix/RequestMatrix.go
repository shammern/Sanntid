package requestmatrix

type RequestMatrix struct {
	hallRequests [][2]bool
	cabRequests  []bool
}

type RequestMatrixDTO struct {
	HallRequests [][2]bool `json:"hallRequests"`
	CabRequests  []bool    `json:"cabRequests"`
}

func NewRequestMatrix(numFloors int) *RequestMatrix {
	rm := &RequestMatrix{
		hallRequests: make([][2]bool, numFloors),
		cabRequests:  make([]bool, numFloors),
	}
	return rm
}

func (rm *RequestMatrix) SetHallRequest(floor int, direction int, active bool) {
	if floor < 0 || floor >= len(rm.hallRequests) {
		return
	}
	if direction < 0 || direction > 1 {
		return
	}
	rm.hallRequests[floor][direction] = active
}

func (rm *RequestMatrix) SetCabRequest(floor int, active bool) {
	if floor < 0 || floor >= len(rm.cabRequests) {
		return
	}
	rm.cabRequests[floor] = active
}

func (rm *RequestMatrix) SetAllHallRequest(orders [][2]bool) {
	rm.hallRequests = orders
}

func (rm *RequestMatrix) SetAllCabRequest(orders []bool) {
	rm.cabRequests = orders
}

func (rm *RequestMatrix) ClearHallRequest(floor int, direction int) {
	rm.SetHallRequest(floor, direction, false)
}

func (rm *RequestMatrix) ClearCabRequest(floor int) {
	rm.SetCabRequest(floor, false)
}

func (rm *RequestMatrix) GetCabRequest() []bool {
	return rm.cabRequests
}

func (rm *RequestMatrix) GetHallRequest() [][2]bool {
	return rm.hallRequests
}

// Extracts data from RM to broadcast on network
func (rm *RequestMatrix) ToDTO() RequestMatrixDTO {
	return RequestMatrixDTO{
		HallRequests: rm.GetHallRequest(),
		CabRequests:  rm.GetCabRequest(),
	}
}
