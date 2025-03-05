package config

//Might change these to JSON 
var ElevatorAddresses = map[int]string{
	1: "localhost:15555",
	2: "localhost:15556",
	3: "localhost:15557",
}

var UDPAddresses = map[int]string{
	1: "127.0.0.1:8001",
	2: "127.0.0.1:8002",
	3: "127.0.0.1:8003",
}

var NumFloors = 4
