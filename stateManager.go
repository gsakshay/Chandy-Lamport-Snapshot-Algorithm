package main

import (
	"fmt"
	"sync"
)

type State struct {
	Counter  int
	HasToken bool
}

type Snapshot struct {
	ID            int
	State         State
	Complete      bool
	BlockedQueues int
	Queues        map[int]*Queue
}

type StateManager struct {
	currentState State
	snapshots    map[int]*Snapshot
	peerManager  *PeerManager
	mu           sync.RWMutex
}

func NewStateManager(peerManager *PeerManager) *StateManager {
	return &StateManager{
		currentState: State{Counter: 0},
		snapshots:    make(map[int]*Snapshot),
		peerManager:  peerManager,
	}
}

func (sm *StateManager) IncrementCounter() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.currentState.Counter++
}

func (sm *StateManager) SetHasToken(value bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.currentState.HasToken = value
}

func (sm *StateManager) GetHasToken() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState.HasToken
}

func (sm *StateManager) GetCurrentState() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState
}

func (sm *StateManager) InitiateSnapshot(snapshotID int) *Snapshot {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	snapshot := &Snapshot{
		ID:       snapshotID,
		State:    sm.currentState,
		Complete: false,
		Queues:   make(map[int]*Queue),
	}

	// Initialize queues for all peers
	peers := sm.peerManager.GetPeers()
	for _, peer := range peers {
		snapshot.Queues[peer.ID] = NewQueue()
	}

	sm.snapshots[snapshotID] = snapshot
	return snapshot
}

func (sm *StateManager) GetSnapshot(snapshotID int) (*Snapshot, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	snapshot, exists := sm.snapshots[snapshotID]
	return snapshot, exists
}

func (sm *StateManager) GetSnapshots() map[int]*Snapshot {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	snapshots := make(map[int]*Snapshot, len(sm.snapshots))
	for id, snapshot := range sm.snapshots {
		snapshots[id] = snapshot
	}

	return snapshots
}

func (sm *StateManager) RecordMessageInSnapshot(snapshotID, senderID int, message interface{}) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	snapshot, exists := sm.snapshots[snapshotID]
	if !exists {
		return fmt.Errorf("snapshot %d does not exist", snapshotID)
	}

	queue, exists := snapshot.Queues[senderID]
	if !exists {
		return fmt.Errorf("queue for sender %d does not exist in snapshot %d", senderID, snapshotID)
	}

	return queue.Append(message)
}

func (sm *StateManager) CloseChannelInSnapshot(snapshotID, senderID int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	snapshot, exists := sm.snapshots[snapshotID]
	if !exists {
		return fmt.Errorf("snapshot %d does not exist", snapshotID)
	}

	queue, exists := snapshot.Queues[senderID]
	if !exists {
		return fmt.Errorf("queue for sender %d does not exist in snapshot %d", senderID, snapshotID)
	}

	queue.SetBlock(true)
	snapshot.BlockedQueues += 1

	return nil
}

func (sm *StateManager) IsSnapshotComplete(snapshotID int, npeers int) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	snapshot, exists := sm.snapshots[snapshotID]
	if !exists {
		return false
	}

	if snapshot.BlockedQueues == npeers {
		return true
	}

	return false
}
