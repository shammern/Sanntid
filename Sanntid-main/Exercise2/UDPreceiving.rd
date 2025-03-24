package main

//Import necesary tools
import (
	"fmt"
	"net"
)

func main() {
	addr := net.UDPAddr{
		Port: 30000,
		IP:   net.ParseIP("0.0.0.0"),
	}

	//Creates an UDP socket using ListenUDP, if unable to set up socket it throws an error.
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}

	//Sets up a safety to terminate connection when function ends
	defer conn.Close()

	//Makes a 1024Byte buffer for receiving messages 
	buffer := make([]byte, 1024)

	//Receives messages on the socket using readdromUDP and writes data to buffer. Throws error if no data is received
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error reading from UDP:", err)
			continue
		}

		fmt.Printf("Received message from %s: %s\n", remoteAddr, string(buffer[:n]))
	}
}