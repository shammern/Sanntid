package main

//Import necesary tools
import (
	"fmt"
	"net"
)

func main() {
	print("--------------------START OF PROGRAM---------------------\n")
	addr := net.UDPAddr{
		Port: 20020,
		//IP:   net.ParseIP("255.255.255.255"), //Broadcast
		IP:   net.ParseIP("10.100.23.204"), //Unicast
	}

	//Creates an UDP socket using DialUDP, if unable to set up socket it throws an error.
	sendingConn, err := net.DialUDP("udp", nil, &addr)
	if err != nil {
		fmt.Println("Error sending:", err)
		return
	}

	//Sets up a safety to terminate connection when function ends
	defer sendingConn.Close()

	
	_, err = sendingConn.Write([]byte("Hello from group 62"))
	if err != nil{
		fmt.Println("Error sending:", err)
	}
	

	//receiving connection
	receivingAddr := net.UDPAddr{
		Port: 20020,
		IP:   net.ParseIP("0.0.0.0"),
		//IP:   net.ParseIP("10.100.23.204"), Illigal activities
	}

	receiveConn, err := net.ListenUDP("udp", &receivingAddr)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}

	buffer := make([]byte, 1024)

	//Receives messages on the socket using readdromUDP and writes data to buffer. Throws error if no data is received
	for {
		n, remoteAddr, err := receiveConn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error reading from UDP:", err)
			continue
		}

		fmt.Printf("Received message from %s: %s\n", remoteAddr, string(buffer[:n]))
	}
	
}