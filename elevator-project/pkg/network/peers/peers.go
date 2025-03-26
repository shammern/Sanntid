package peers

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/network/conn"
	"elevator-project/pkg/state"
	"fmt"
	"net"
	"sort"
	"strconv"
	"time"
)

var Ch_peerLost = make(chan []int, 2)

type PeerUpdate struct {
	Peers []string
	New   string
	Lost  []string
}

var LatestPeerUpdate PeerUpdate

func Transmitter(port int, id string, transmitEnable <-chan bool) {

	conn := conn.DialBroadcastUDP(port)
	addr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", port))

	enable := true
	for {
		select {
		case enable = <-transmitEnable:
		case <-time.After(config.HeartBeatInterval):
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

		conn.SetReadDeadline(time.Now().Add(config.HeartBeatInterval))
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
			if time.Now().Sub(v) > config.Timeout {
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

// Monitors the network and triggers events when a peer is disconnects/connects
func P2Pmonitor(stateStore *state.Store) {
	ch_peerUpdate := make(chan PeerUpdate)
	ch_peerTxEnable := make(chan bool)
	go Transmitter(config.P2Pport, strconv.Itoa(config.ElevatorID), ch_peerTxEnable)
	go Receiver(config.P2Pport, ch_peerUpdate)

	for {
		update := <-ch_peerUpdate
		LatestPeerUpdate = update
		fmt.Printf("Peer update:\n")
		fmt.Printf("  Peers:    %q\n", update.Peers)
		fmt.Printf("  New:      %q\n", update.New)
		fmt.Printf("  Lost:     %q\n", update.Lost)

		// Mark lost peers as unavailable for new orders, but skip our own elevator id.
		for _, lostPeer := range update.Lost {
			if lostPeer == strconv.Itoa(config.ElevatorID) {
				continue
			}
			if peerID, err := strconv.Atoi(lostPeer); err == nil {
				stateStore.UpdateElevatorAvailability(peerID, false)
			}
		}

		if len(update.Lost) > 0 {
			var lostIDs []int
			for _, lostPeer := range update.Lost {
				if id, err := strconv.Atoi(lostPeer); err == nil {
					lostIDs = append(lostIDs, id)
				}

				// Sends lost units to ackMonitor
				Ch_peerLost <- lostIDs
			}
		}
	}
}
