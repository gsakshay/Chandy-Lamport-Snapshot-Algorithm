# Chandi-Lamport Algorithm Implementation Report

## Overview

The Chandi-Lamport Algorithm is a distributed algorithm for recording a consistent global snapshot of the state of a distributed system. The algorithm is especially useful for detecting deadlocks, analyzing distributed processes, or performing distributed debugging.

This specific implementation in Go follows the core principles of the algorithm, focusing on the snapshot capturing of the process state and channel states between them.

## Key Algorithmic Logic

### Token Passing Logic

1. Each process either starts with the token (if it is the starter) or waits for a token to arrive.
2. Upon receiving a token, the process increments its state (counter) and logs this state.
3. If a snapshot is triggered (when the process reaches a specific state), the process sends a marker to all its outgoing channels.
4. The token is passed to the next peer, allowing for continued token circulation.

### Snapshotting Logic

1. When a process receives a marker for the first time, it:
   - Records its current state.
   - Closes the incoming channel on which the marker was received.
   - Sends markers to all outgoing channels.
2. The process then records any messages that arrive on its remaining open channels.
3. When all channels are closed (i.e., markers have been received from all peers), the snapshot is complete.


## Application Flow

This implementation integrates the Chandi-Lamport Algorithm within a distributed token-passing system. The key sections of the code are:

- **Initialization and Connection Establishment**: The application reads configuration values, such as the host file, debugging flags, snapshot triggers, etc.
- **Pass Token Process**: The processes are interconnected in the form of a ring and pass a token among themselves.
- **Snapshotting Process**: When triggered, the snapshotting process captures the state of the system.

## Detailed Breakdown

### 1. Initialization and Connection Establishment

#### 1.1 Configurations

The configuration is handled using the `Config` struct, which stores parameters like TCP ports, delay values, snapshot identifiers, etc. 

```go
var config Config

func parseFlagsAndAssignConstants() { ... }
```

The system reads configuration details and the hostname of the current process. The process's role (whether it is a starter or participant) is also defined here.

#### 1.2. Reading the Hosts File

```go
func readHostsFile(filePath string) ([]string, error) { ... }
```

The `readHostsFile` function reads the list of peers from a file. These peers are later used to establish connections between processes.

#### 1.3. Peer Manager Initialization

```go
func initializePeerManager() *PeerManager { ... }
```

The `PeerManager` keeps track of each process (peer) in the system. Each process is assigned an ID based on the order in the hosts file. Additionally, each process knows its predecessor and successor in the token-passing cycle.

- **Predecessor**: The peer that sends a token to this process.
- **Successor**: The peer to which this process sends a token.

#### 1.4. TCP Connection Establishment

```go
func establishConnections(peerManager *PeerManager, connectionsEstablished chan bool) { ... }
```

This function is responsible for establishing TCP connections between the processes and stores in PeerManager for further messaging. 

```go
go func() {
    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Printf("Error accepting connection: %v\n", err)
            continue
        }
        go HandleConnection(conn, messageReceiverCh)
    }
}()
```

The application listens for incoming connections on a TCP port and spawns a goroutine to handle each incoming connection. Received messages are parsed and placed on the appropriate channel (`tokenMessageCh` and `markerMessageCh`).

### 2. Token Passing

```go
func passToken(stateManager *StateManager, peerManager *PeerManager, tokenMessages chan *Message, markerMessageCh chan *Message, ctx context.Context) { ... }
```

This function is responsible for the core token-passing logic. Here’s how it works:

- A process receives the token message on its incoming channel.
- It increments its local state (counter) and checks if the snapshot needs to be triggered.
- The process logs its current state and forwards the token to the next process in the ring after an optional delay.

This function also interacts with the `StateManager` to record messages received during an ongoing snapshot.

### 3. Snapshotting Algorithm

```go
func snapshotting(stateManager *StateManager, peerManager *PeerManager, markerMessages chan *Message, ctx context.Context) { ... }
```

The snapshotting process is responsible for taking consistent global snapshots. The workflow is as follows:

- When a marker is received, the process checks if it is the first time receiving a marker for a particular snapshot.
- If it’s the first marker, the process records its state and sends markers to all its outgoing channels.
- The process then starts recording any messages received on its incoming channels.
- Once markers are received on all channels (i.e., all channels are closed), the snapshot is marked as complete.

This function interacts heavily with the `StateManager` to track snapshots.

## Supplementary Files

### 1. **`peerManager.go`**
   - Manages peer connections, assigning IDs, and storing details about each peer (hostname, address, etc.).
   - Offers functionality for setting neighbors, managing TCP connections, and handling peer-related queries.
   - Core responsibilities: Peer management, connection setup, and neighbor relationships.

### 2. **`queue.go`**
   - Implements a thread-safe FIFO queue with blocking capabilities.
   - Used to store messages between processes and manage inflight messages in channels during snapshots.
   - Core responsibilities: Message queueing and blocking channels when a snapshot is in progress.

### 3. **`stateManager.go`**
   - Manages the state of the current process (token, counter) and handles snapshot operations.
   - Stores and manages snapshots, including recording incoming messages, closing channels, and determining snapshot completeness.
   - Core responsibilities: State management and snapshot coordination.

### 4. **`tcpCommunication.go`**
   - Handles sending and receiving messages over TCP.
   - Supports JSON serialization/deserialization for messages and markers.
   - Core responsibilities: Network communication, message transmission, and handling incoming connections.

These files provide the necessary components and utilities to support the main `Chandi-Lamport` algorithm implementation, focusing on peer management, queue handling, state/snapshot management, and TCP communication.

### Log Format

The application logs the following information:

1. **Process ID**: The ID of the process executing the log.
2. **State Information**: The current state (counter) of the process.
3. **Token Information**: Who received/sent the token.
4. **Snapshot Information**: The state of snapshots, such as when a channel is closed or when a snapshot is complete.

Sample log entry:
```plaintext
2024-10-05 20:36:51 2024/10/06 00:36:51 {proc_id: 1, state: 0, predecessor: 5, successor: 2}
2024-10-05 20:36:51 2024/10/06 00:36:51 {proc_id: 1,  state: 1}
2024-10-05 20:36:51 2024/10/06 00:36:51 {proc_id: 1,  sender: 1, receiver: 2, message: "token"}
2024-10-05 20:36:52 2024/10/06 00:36:52 {proc_id: 1,  sender: 5, receiver: 1, message: "token"}
2024-10-05 20:36:52 2024/10/06 00:36:52 {proc_id: 1,  state: 2}
2024-10-05 20:36:52 2024/10/06 00:36:52 {proc_id:1, snapshot_id: 1, snapshot:"started"}
```

### Handling Snapshot Completeness

A snapshot is complete when all incoming channels have been closed, i.e., markers have been received from all connected processes. The `StateManager` monitors the state of each snapshot and logs when a snapshot is finished.
