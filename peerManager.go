package main

import (
	"fmt"
	"net"
	"sync"
)

type Peer struct {
	ID       int
	Hostname string
	Address  string
	Conn     net.Conn
}

type PeerManager struct {
	peers     map[int]Peer
	peerCount int
	selfID    int
	prevID    int
	nextID    int
	mu        sync.RWMutex
}

func NewPeerManager() *PeerManager {
	return &PeerManager{
		peers: make(map[int]Peer),
	}
}

func (pm *PeerManager) AddPeer(id int, hostname, address string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	peer := Peer{ID: id, Hostname: hostname, Address: address}
	pm.peers[id] = peer
	pm.peerCount += 1
}

func (pm *PeerManager) GetPeers() []Peer {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	peers := make([]Peer, 0, len(pm.peers))
	for _, peer := range pm.peers {
		if peer.ID != pm.selfID {
			peers = append(peers, peer)
		}
	}
	return peers
}

func (pm *PeerManager) GetPeerCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.peerCount
}

func (pm *PeerManager) GetPeer(id int) (Peer, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	peer, ok := pm.peers[id]
	return peer, ok
}

func (pm *PeerManager) SetNeighbors(prevID, nextID int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.prevID = prevID
	pm.nextID = nextID
}

func (pm *PeerManager) GetNeighbors() (prevID, nextID int) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.prevID, pm.nextID
}

func (pm *PeerManager) SetSelf(selfID int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.selfID = selfID
}

func (pm *PeerManager) GetSelfID() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.selfID
}

func (pm *PeerManager) GetSelf() (Peer, bool) {
	return pm.GetPeer(pm.GetSelfID())
}

func (pm *PeerManager) GetPrevPeer() (Peer, bool) {
	prevID, _ := pm.GetNeighbors()
	return pm.GetPeer(prevID)
}

func (pm *PeerManager) GetNextPeer() (Peer, bool) {
	_, nextID := pm.GetNeighbors()
	return pm.GetPeer(nextID)
}

func (pm *PeerManager) SetConnection(hostPeerId int, conn net.Conn) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("connect object is empty")
	}

	if peer, ok := pm.peers[hostPeerId]; ok {
		peer.Conn = conn
		pm.peers[peer.ID] = peer
		return nil
	}

	return fmt.Errorf("Peer with ID %d not found", hostPeerId)
}
