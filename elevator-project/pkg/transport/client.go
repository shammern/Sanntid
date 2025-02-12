// File: pkg/transport/client.go
package transport

import (
	"elevator-project/pkg/message"
	"encoding/json"
	"fmt"
	"net"
)

// SendUDPMessage sends a JSON-encoded message to a remote UDP address.
// remoteAddr is a string in the form "IP:port" (e.g., "192.168.1.100:8000").
// msg is the message to be sent.
func SendUDPMessage(remoteAddr string, msg message.Message) error {
	// Resolve the remote UDP address.
	udpAddr, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		return fmt.Errorf("failed to resolve remote address %s: %w", remoteAddr, err)
	}

	// Dial the remote UDP address.
	// We don't need to specify a local address; nil lets the system choose.
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return fmt.Errorf("failed to dial remote address %s: %w", remoteAddr, err)
	}
	defer conn.Close()

	// Marshal the message into JSON.
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Append a newline if your receiver uses newline-delimited messages.
	// This should match the framing used in your UDP listener.
	msgBytes = append(msgBytes, '\n')

	// Write the JSON message to the UDP connection.
	_, err = conn.Write(msgBytes)
	if err != nil {
		return fmt.Errorf("failed to send message to %s: %w", remoteAddr, err)
	}

	fmt.Printf("Sent message to %s\n", remoteAddr)
	return nil
}
