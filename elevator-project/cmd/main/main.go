package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// Message types.
type MessageType int

const (
	Heartbeat MessageType = iota
	Promotion
)

// Message is sent over UDP.
type Message struct {
	Type       MessageType `json:"type"`
	ElevatorID int         `json:"elevator_id"`
	// For Heartbeat messages, Role is the sender’s current role.
	// For Promotion messages, Role tells the recipient which role to assume.
	Role      string    `json:"role"`
	MasterID  int       `json:"master_id,omitempty"` // Indicates current master
	Timestamp time.Time `json:"timestamp"`
}

// PeerStatus holds the last known heartbeat and role of a peer.
type PeerStatus struct {
	LastHeartbeat time.Time
	Role          string
}

// Elevator represents one node in the system.
type Elevator struct {
	id                     int
	role                   string // "master", "backup", or "idle"
	hbInterval             time.Duration
	hbTimeout              time.Duration
	peers                  []string           // List of peer UDP addresses
	lastHeartbeat          map[int]PeerStatus // Keyed by ElevatorID
	conn                   *net.UDPConn
	backupSince            time.Time
	currentMaster          int
	lastPromotionBroadcast time.Time // When the last promotion broadcast was sent
}

func NewElevator(id int, role, listenAddr string, peers []string) *Elevator {
	udpAddr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		fmt.Println("Error resolving listen address:", err)
		os.Exit(1)
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Error listening on UDP address:", err)
		os.Exit(1)
	}
	cm := 0
	if role == "master" {
		cm = id
	}
	return &Elevator{
		id:            id,
		role:          role,
		hbInterval:    1 * time.Second,
		hbTimeout:     3 * time.Second,
		peers:         peers,
		lastHeartbeat: make(map[int]PeerStatus),
		conn:          conn,
		currentMaster: cm,
	}
}

// sendHeartbeat broadcasts a heartbeat to all peers.
func (e *Elevator) sendHeartbeat() {
	ticker := time.NewTicker(e.hbInterval)
	defer ticker.Stop()
	for range ticker.C {
		msg := Message{
			Type:       Heartbeat,
			ElevatorID: e.id,
			Role:       e.role,
			Timestamp:  time.Now(),
		}
		// If master, include its own id.
		if e.role == "master" {
			msg.MasterID = e.id
		}
		data, err := json.Marshal(msg)
		if err != nil {
			fmt.Println("Error marshaling heartbeat:", err)
			continue
		}
		for _, peerAddrStr := range e.peers {
			peerAddr, err := net.ResolveUDPAddr("udp", peerAddrStr)
			if err != nil {
				fmt.Println("Error resolving peer address:", err)
				continue
			}
			_, err = e.conn.WriteToUDP(data, peerAddr)
			if err != nil {
				fmt.Println("Error sending heartbeat to", peerAddrStr, ":", err)
			}
		}
		fmt.Printf("Elevator %d (%s) sent heartbeat at %v\n", e.id, e.role, msg.Timestamp.Format("15:04:05"))
	}
}

// listen continuously receives UDP messages and updates peer statuses.
func (e *Elevator) listen() {
	buf := make([]byte, 1024)
	for {
		n, _, err := e.conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error reading UDP:", err)
			continue
		}
		var msg Message
		err = json.Unmarshal(buf[:n], &msg)
		if err != nil {
			fmt.Println("Error unmarshaling message:", err)
			continue
		}
		// Ignore our own messages.
		if msg.ElevatorID == e.id {
			continue
		}

		switch msg.Type {
		case Heartbeat:
			// Update last heartbeat.
			e.lastHeartbeat[msg.ElevatorID] = PeerStatus{
				LastHeartbeat: msg.Timestamp,
				Role:          msg.Role,
			}
			fmt.Printf("Elevator %d (%s) received heartbeat from Elevator %d (%s) at %v\n",
				e.id, e.role, msg.ElevatorID, msg.Role, msg.Timestamp.Format("15:04:05"))
			// If a heartbeat indicates master, update currentMaster.
			if msg.Role == "master" {
				e.currentMaster = msg.ElevatorID
			}
		case Promotion:
			// Idle elevators change to backup upon receiving a promotion message.
			if e.role == "idle" {
				fmt.Printf("Elevator %d (%s) received promotion message, becoming backup\n", e.id, e.role)
				e.role = "backup"
				e.backupSince = time.Now()
				e.currentMaster = msg.MasterID
			}
		}
	}
}

// broadcastPromotion sends a promotion message to all peers.
func (e *Elevator) broadcastPromotion() {
	msg := Message{
		Type:       Promotion,
		ElevatorID: e.id,
		Role:       "backup", // Instruct idle recipients to become backup.
		Timestamp:  time.Now(),
		MasterID:   e.id, // Master broadcasts its own id.
	}
	data, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Error marshaling promotion message:", err)
		return
	}
	for _, peerAddrStr := range e.peers {
		peerAddr, err := net.ResolveUDPAddr("udp", peerAddrStr)
		if err != nil {
			fmt.Println("Error resolving peer address for promotion:", err)
			continue
		}
		_, err = e.conn.WriteToUDP(data, peerAddr)
		if err != nil {
			fmt.Println("Error sending promotion message to", peerAddrStr, ":", err)
		} else {
			fmt.Printf("Elevator %d (%s) broadcasted promotion message at %v\n", e.id, e.role, time.Now().Format("15:04:05"))
		}
	}
}

// monitorPeers checks for stale heartbeats and triggers role promotions.
// Backups check for missing master's heartbeat.
// Masters check for missing backups and broadcast promotion only once per cooldown period.
func (e *Elevator) monitorPeers() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	gracePeriod := 2 * time.Second
	promotionCooldown := 5 * time.Second
	for range ticker.C {
		now := time.Now()
		// For backups: check master's heartbeat.
		if e.role == "backup" && e.currentMaster != e.id {
			status, ok := e.lastHeartbeat[e.currentMaster]
			if !ok || now.Sub(status.LastHeartbeat) > e.hbTimeout {
				if time.Since(e.backupSince) > gracePeriod {
					fmt.Printf("Elevator %d (%s) promoting itself to master due to missing master (ID %d)\n", e.id, e.role, e.currentMaster)
					e.role = "master"
					e.currentMaster = e.id
					e.broadcastPromotion()
				}
			}
		}
		// For masters: check for missing backup(s).
		if e.role == "master" {
			for peerID, status := range e.lastHeartbeat {
				if status.Role == "backup" && now.Sub(status.LastHeartbeat) > e.hbTimeout {
					if now.Sub(e.lastPromotionBroadcast) > promotionCooldown {
						fmt.Printf("Elevator %d (%s) detected missing backup (ID %d), promoting an idle elevator\n", e.id, e.role, peerID)
						e.broadcastPromotion()
						e.lastPromotionBroadcast = now
					}
				}
			}
		}
	}
}

func main() {
	// Command-line arguments:
	// -id: Elevator ID.
	// -role: "master", "backup", or "idle".
	// -listen: UDP address for this elevator (e.g., "127.0.0.1:8001").
	// -peers: Comma-separated list of peer UDP addresses.
	id := flag.Int("id", 0, "Elevator ID")
	role := flag.String("role", "idle", "Elevator role: master, backup, or idle")
	listenAddr := flag.String("listen", "", "Listen UDP address (e.g., 127.0.0.1:8001)")
	peersStr := flag.String("peers", "", "Comma-separated list of peer UDP addresses")
	flag.Parse()

	if *id == 0 || *listenAddr == "" {
		fmt.Println("Usage: -id=<id> -role=<role> -listen=<address> -peers=<peer1,peer2,...>")
		os.Exit(1)
	}

	peers := []string{}
	if *peersStr != "" {
		peers = strings.Split(*peersStr, ",")
	}

	elevator := NewElevator(*id, *role, *listenAddr, peers)

	go elevator.sendHeartbeat()
	go elevator.listen()
	go elevator.monitorPeers()

	select {}
}
