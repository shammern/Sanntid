package transport

import (
	"fmt"
	"net"
	"time"
	"elevator-project/pkg/message"
)

func SendMessage(msg message.Message, peerAddr string) error {
	addr, err := net.ResolveUDPAddr("udp", peerAddr)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	data, err := message.Marshal(msg)
	if err != nil {
		return err
	}

	switch msg.Type {
	case message.Order:
		//Max retries will have to be tuned when testing with packetloss to find a suitable amount.
		//Might also be implemented as a loop that breaks when an ack is received.
		maxRetries := 3
		for i := 0; i < maxRetries; i++ {
			_, err = conn.Write(data)
			if err != nil {
				fmt.Println("Error sending order message:", err)
			} else {
				fmt.Printf("Order message sent to %s, attempt %d\n", peerAddr, i+1)
			}
			
			time.Sleep(10 * time.Millisecond)
		}
	default:
		_, err = conn.Write(data)
		if err != nil {
			return err
		}
	}

	return nil
}