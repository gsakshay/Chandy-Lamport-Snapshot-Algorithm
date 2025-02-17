package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type Config struct {
	TCPPort       string
	Debug         bool
	Starter       bool
	HostsFile     string
	TokenDelay    float64
	MarkerDelay   float64
	StateSnapshot int
	SnapshotId    int
	Hostname      string
	TokenMessage  string
	MarkerMessage string
	Timeout       time.Duration // Final timeout for the program to exit
}

var config Config

func parseFlagsAndAssignConstants() {
	flag.StringVar(&config.HostsFile, "h", "", "Path to hosts file")
	flag.BoolVar(&config.Debug, "d", false, "Enable debugging mode")
	flag.BoolVar(&config.Starter, "x", false, "Token")
	flag.Float64Var(&config.TokenDelay, "t", 0.0, "Token Delay")
	flag.Float64Var(&config.MarkerDelay, "m", 0.0, "Marker Delay")
	flag.IntVar(&config.StateSnapshot, "s", 0, "Initiate Snapshot once reached this State")
	flag.IntVar(&config.SnapshotId, "p", 0, "Initiate Snapshot once reached this State")
	flag.Parse()

	config.TCPPort = "8888"
	config.Timeout = 5 * time.Minute
	config.TokenMessage = "token"
	config.MarkerMessage = "marker"

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Error getting hostname: %v", err)
	}
	config.Hostname = hostname
}

func readHostsFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var peers []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			peers = append(peers, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return peers, nil
}

func initializePeerManager() *PeerManager {
	spm := NewPeerManager()
	peers, err := readHostsFile(config.HostsFile)
	if err != nil {
		log.Fatalf("error reading hosts file: %v", err)
	}

	peerCount := len(peers)

	for i, hostname := range peers {
		address := fmt.Sprintf("%s:%s", hostname, config.TCPPort)
		if hostname == config.Hostname {
			spm.SetSelf(i + 1)
		} else { // Only add others from the hostfile
			spm.AddPeer(i+1, hostname, address)
		}

	}
	selfIndex := spm.GetSelfID()
	if selfIndex == 0 {
		log.Fatalf("hostname not found in hosts file")
	}

	prevID := ((selfIndex - 2 + peerCount) % peerCount) + 1
	nextID := selfIndex%peerCount + 1

	spm.SetNeighbors(prevID, nextID)
	return spm
}

func passToken(stateManager *StateManager, peerManager *PeerManager, tokenMessages chan *Message, markerMessageCh chan *Message, ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-tokenMessages: // Wait and pass the token
			stateManager.SetHasToken(true)
			selfId := peerManager.GetSelfID()

			stateManager.IncrementCounter()
			currentStateCounter := stateManager.GetCurrentState().Counter

			if msg.Header.SenderID != 0 { // Dont print for the starter
				log.Printf("{proc_id: %v,  sender: %v, receiver: %v, message: \"token\"}\n",
					selfId, msg.Header.SenderID, selfId)
				log.Printf("{proc_id: %v,  state: %v}\n", selfId, currentStateCounter)

			}

			if config.StateSnapshot != 0 && currentStateCounter == config.StateSnapshot {
				payload := make([]byte, 4)
				binary.BigEndian.PutUint32(payload, uint32(config.SnapshotId))
				markerMessageCh <- &Message{
					Header: MessageHeader{
						SenderID:    0,
						MessageType: MARKER,
						PayloadSize: int32(len(payload)),
					},
					Payload: payload,
				}
			}

			// Add to queue if necessary
			for id, snapshot := range stateManager.GetSnapshots() {
				if !snapshot.Complete && !snapshot.Queues[int(msg.Header.SenderID)].IsBlocked() {
					stateManager.RecordMessageInSnapshot(id, int(msg.Header.SenderID), "token")
				}
			}

			time.Sleep(time.Duration(config.TokenDelay * float64(time.Second))) // Sleep - Mimic for some work

			if peer, ok := peerManager.GetNextPeer(); ok {
				err := SendToken(peer.Conn, selfId)
				if err != nil {
					log.Printf("Failed to send Token Message: %v\n", err.Error())
				} else {
					stateManager.SetHasToken(false)
					log.Printf("{proc_id: %v,  sender: %v, receiver: %v, message: \"token\"}\n",
						selfId, selfId, peer.ID)
				}
			}
		}
	}
}

func snapshotting(stateManager *StateManager, peerManager *PeerManager, markerMessages chan *Message, ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-markerMessages:
			selfId := peerManager.GetSelfID()
			snapshotID, err := ParseMarkerPayload(msg.Payload)
			if err != nil {
				log.Printf("Failed to parse Marker Message: %v\n", err.Error())
			}
			if snap, ok := stateManager.GetSnapshot(snapshotID); ok {
				// Block the channel for the sender
				err := stateManager.CloseChannelInSnapshot(snapshotID, int(msg.Header.SenderID))
				if err == nil {
					log.Printf("{proc_id:%v, snapshot_id: %v, snapshot:\"channel closed\", channel:%v-%v, queue:[%v]}\n",
						selfId, snapshotID, int(msg.Header.SenderID), selfId, snap.Queues[int(msg.Header.SenderID)].GetCommaSepratedValues())
				}
				// Check if all the channels are closed
				if stateManager.IsSnapshotComplete(snap.ID, peerManager.GetPeerCount()) {
					log.Printf("{proc_id:%v, snapshot_id: %v, snapshot:\"complete\"}\n",
						selfId, snapshotID)
				}
			} else {
				// Start the snapshot
				snap := stateManager.InitiateSnapshot(snapshotID)
				log.Printf("{proc_id:%v, snapshot_id: %v, snapshot:\"started\"}", selfId, snapshotID)

				// Block the channel for the sender
				err := stateManager.CloseChannelInSnapshot(snapshotID, int(msg.Header.SenderID))
				if err == nil {
					log.Printf("{proc_id:%v, snapshot_id: %v, snapshot:\"channel closed\", channel:%v-%v, queue:[]}\n",
						selfId, snapshotID, int(msg.Header.SenderID), selfId)
				}

				time.Sleep(time.Duration(config.MarkerDelay * float64(time.Second))) // Sleep - Mimic for some work
				// Send marker to all peers
				for _, peer := range peerManager.GetPeers() {
					err := SendMarker(peer.Conn, selfId, snapshotID)
					if err != nil {
						log.Printf("Failed to send Marker Message: %v\n", err.Error())
					} else {
						hasToken := "NO"
						if stateManager.GetHasToken() {
							hasToken = "YES"
						}
						log.Printf("{proc_id:%v, snapshot_id: %v, sender:%v, receiver:%v, msg:\"marker\", state:%v, has_token:%v}\n",
							selfId, snapshotID, selfId, peer.ID, snap.State.Counter, hasToken)
					}
				}

			}
		}
	}
}

func establishConnections(peerManager *PeerManager, connectionsEstablished chan bool) {
	ticker := time.NewTicker(10 * time.Second) // Try establishing connections for 10 seconds
	for {
		select {
		case <-ticker.C:
			return
		default:
			errors := 0
			for _, peer := range peerManager.GetPeers() {
				if peer.Conn == nil {
					conn, err := net.Dial("tcp", peer.Address)
					connErr := peerManager.SetConnection(peer.ID, conn)
					if err != nil || connErr != nil {
						errors += 1
					}

				}
			}
			if errors == 0 {
				connectionsEstablished <- true
				return
			}
		}

	}
}

func main() {
	parseFlagsAndAssignConstants()
	peerManager := initializePeerManager()
	stateManager := NewStateManager(peerManager)

	prevId, nextId := peerManager.GetNeighbors()
	log.Printf("{proc_id: %v, state: %v, predecessor: %v, successor: %v}\n",
		peerManager.GetSelfID(), stateManager.GetCurrentState().Counter, prevId, nextId)

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", config.Hostname, config.TCPPort))
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	// Establish connections
	connectionsEstablished := make(chan bool)
	go establishConnections(peerManager, connectionsEstablished)

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	messageReceiverCh := make(chan *Message, 100)
	tokenMessageCh := make(chan *Message, 100)
	markerMessageCh := make(chan *Message, 100)

	go func() { // Keep listening for connections
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("Error accepting connection: %v\n", err)
				continue
			}
			go HandleConnection(conn, messageReceiverCh)
		}
	}()

	// Pass token
	go passToken(stateManager, peerManager, tokenMessageCh, markerMessageCh, ctx)
	// Shapshot Algorithm - Chandi Lamport
	go snapshotting(stateManager, peerManager, markerMessageCh, ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down...")
			return
		case <-connectionsEstablished: // Wait for the connections to get established before sending any messages
			if config.Starter { // Start passing the token
				tokenMessageCh <- &Message{
					Header: MessageHeader{
						SenderID:    0,
						MessageType: TOKEN,
						PayloadSize: 0,
					},
					Payload: nil,
				}
			}
		case msg := <-messageReceiverCh:
			switch msg.Header.MessageType {
			case TOKEN:
				tokenMessageCh <- msg
			case MARKER:
				markerMessageCh <- msg
			default:
				log.Printf("Received unknown message type: %T\n", msg.Header.MessageType)
			}
		}
	}
}
