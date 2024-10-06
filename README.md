# Chandi-Lamport Distributed Snapshot Algorithm

## Overview

This project implements the **Chandi-Lamport Algorithm** for recording a consistent global snapshot in a distributed system. The algorithm is applied within a token-passing system, where processes communicate over TCP and record their state and channel messages during snapshots.

### Key Features
- Distributed snapshot capturing using marker messages.
- Token passing among multiple processes.
- Configurable delays for token propagation and marker transmission.

## Requirements

- Docker
- Docker Compose

## Usage

1. Clone the repository and navigate to the implementation.

2. Build the Docker image:
   ```
   docker build -t prj2 .
   ```

3. Run the program using Docker Compose:
   ```
   docker-compose -f docker-compose-testcase-[1-5*].yml up
   ```

   This will start 5 peer containers as defined in the `docker-compose-testcase-[1-5*].yml` file.

4. To change the number of peers, token and marker compositions, modify the `docker-compose-testcase-[1-5*].yml` file as needed.

5. To stop the program:
   ```
   docker-compose -f docker-compose-testcase-[1-5*].yml down
   ```

For more detailed information about the implementation, please refer to the REPORT.md file.

## Project Files

- **`main.go`**: The main file running the token passing and snapshotting logic.
- **`peerManager.go`**: Manages peer connections and their relationships.
- **`queue.go`**: Implements the FIFO queues for message handling between processes.
- **`stateManager.go`**: Manages process states and snapshots.
- **`tcpCommunication.go`**: Handles TCP communication and message transmission between peers.
- **`hostsfile.txt`**: List of peer hostnames involved in the distributed system.
- **`Dockerfile`**: Defines the Docker image for the program.
- **`docker-compose-testcase-[1-5*].yml`**: Defines the multi-container Docker application.