package peers

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/message"
	"elevator-project/pkg/network/conn"
	"elevator-project/pkg/state"
	"fmt"
	"net"
	"sort"
	"strconv"
	"time"
)

type PeerUpdate struct {
	Peers []string
	New   string
	Lost  []string
}

var LatestPeerUpdate PeerUpdate

const interval = 15 * time.Millisecond
const timeout = 500 * time.Millisecond

func Transmitter(port int, id string, transmitEnable <-chan bool) {

	conn := conn.DialBroadcastUDP(port)
	addr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", port))

	enable := true
	for {
		select {
		case enable = <-transmitEnable:
		case <-time.After(interval):
		}
		if enable {
			conn.WriteTo([]byte(id), addr)
		}
	}
}

func Receiver(port int, peerUpdateCh chan<- PeerUpdate) {

	var buf [1024]byte
	var p PeerUpdate
	lastSeen := make(map[string]time.Time)

	conn := conn.DialBroadcastUDP(port)

	for {
		updated := false

		conn.SetReadDeadline(time.Now().Add(interval))
		n, _, _ := conn.ReadFrom(buf[0:])

		id := string(buf[:n])

		// Adding new connection
		p.New = ""
		if id != "" {
			if _, idExists := lastSeen[id]; !idExists {
				p.New = id
				updated = true
			}

			lastSeen[id] = time.Now()
		}

		// Removing dead connection
		p.Lost = make([]string, 0)
		for k, v := range lastSeen {
			if time.Now().Sub(v) > timeout {
				updated = true
				p.Lost = append(p.Lost, k)
				delete(lastSeen, k)
			}
		}

		// Sending update
		if updated {
			p.Peers = make([]string, 0, len(lastSeen))

			for k, _ := range lastSeen {
				p.Peers = append(p.Peers, k)
			}

			sort.Strings(p.Peers)
			sort.Strings(p.Lost)
			peerUpdateCh <- p
		}
	}
}

func P2Pmonitor(stateStore *state.Store, msgTx chan message.Message) {
	// This function triggers events when elevators join or leave the network.
	peerUpdateCh := make(chan PeerUpdate)
	peerTxEnable := make(chan bool)
	go Transmitter(config.P2Pport, strconv.Itoa(config.ElevatorID), peerTxEnable)
	go Receiver(config.P2Pport, peerUpdateCh)

	for {
		update := <-peerUpdateCh
		LatestPeerUpdate = update
		fmt.Printf("Peer update:\n")
		fmt.Printf("  Peers:    %q\n", update.Peers)
		fmt.Printf("  New:      %q\n", update.New)
		fmt.Printf("  Lost:     %q\n", update.Lost)

		// Mark all peers in the Peers list as available.
		for _, peerStr := range update.Peers {
			if peerID, err := strconv.Atoi(peerStr); err == nil {
				stateStore.UpdateElevatorAvailability(peerID, true)
				fmt.Printf("Marked elevator %d as available\n", peerID)
			} else {
				fmt.Printf("Error parsing peer id %s: %v\n", peerStr, err)
			}
		}

		// Mark lost peers as unavailable, but skip our own elevator id.
		for _, lostPeer := range update.Lost {
			// Skip if the lost peer is the local elevator.
			if lostPeer == strconv.Itoa(config.ElevatorID) {
				continue
			}
			if peerID, err := strconv.Atoi(lostPeer); err == nil {
				stateStore.UpdateElevatorAvailability(peerID, false)
				fmt.Printf("Marked elevator %d as unavailable\n", peerID)
			} else {
				fmt.Printf("Error parsing lost peer id %s: %v\n", lostPeer, err)
			}
		}

		// Ensure our own elevator is always marked as available.
		stateStore.UpdateElevatorAvailability(config.ElevatorID, true)
	}
}
