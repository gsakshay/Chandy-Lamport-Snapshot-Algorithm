FROM golang:1.17-alpine

WORKDIR /app

COPY . .

RUN go build -o chandyLamport main.go peerManager.go tcpCommunication.go queue.go stateManager.go

ENTRYPOINT ["/app/chandyLamport"]