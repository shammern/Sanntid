package transport

//Contains code for sending and receiving JSON messages over UDP, with room to support TCP later.

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// Message defines the structure of the JSON messages that the UDP listener expects.
// Adjust the Payload type as needed for your application.
type Message struct {
	Type      string          `json:"type"`      
	Seq       uint64          `json:"seq"`       
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// StartUDPListener creates a UDP listener on the specified address (e.g., ":8000").
// It continuously reads incoming UDP packets, unmarshals them as JSON, and prints the parsed message.
func StartUDPListener(address string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return fmt.Errorf("error resolving address %s: %v", address, err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("error listening on address %s: %v", address, err)
	}
	defer conn.Close()

	fmt.Printf("UDP server listening on %s\n", address)

	// Create a buffer to store incoming data.
	// Adjust the size if you expect larger messages.
	buf := make([]byte, 4096)

	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Printf("Error reading from UDP: %v\n", err)
			continue
		}

		// Log the details about the received packet.
		fmt.Printf("Received %d bytes from %s\n", n, remoteAddr.String())

		// Unmarshal the JSON data into a Message struct.
		var msg Message
		if err := json.Unmarshal(buf[:n], &msg); err != nil {
			fmt.Printf("Error unmarshaling JSON message: %v\n", err)
			continue
		}

		// Process or log the message.
		// In this example, we simply print the message fields.
		fmt.Printf("Received message:\n")
		fmt.Printf("  Type: %s\n", msg.Type)
		fmt.Printf("  Seq: %d\n", msg.Seq)
		fmt.Printf("  Timestamp: %s\n", msg.Timestamp.Format(time.RFC3339))
		fmt.Printf("  Payload: %s\n", string(msg.Payload))
	}

	// This function never reaches this point because of the infinite loop,
	// but it's here to satisfy the function signature.
	// return nil
}
