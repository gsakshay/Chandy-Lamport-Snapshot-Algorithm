package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

const (
	TOKEN  = 1
	MARKER = 2
)

type MessageHeader struct {
	SenderID    int32
	MessageType int32
	PayloadSize int32
}

type Message struct {
	Header  MessageHeader
	Payload []byte
}

type Marker struct {
	SnapshotID int32
}

func SendMessage(conn net.Conn, senderID int, messageType int, payload []byte) error {
	if conn == nil {
		return errors.New("connection is nil")
	}

	header := MessageHeader{
		SenderID:    int32(senderID),
		MessageType: int32(messageType),
		PayloadSize: int32(len(payload)),
	}

	err := binary.Write(conn, binary.BigEndian, &header)
	if err != nil {
		return fmt.Errorf("error writing header: %v", err)
	}

	_, err = conn.Write(payload)
	if err != nil {
		return fmt.Errorf("error writing payload: %v", err)
	}

	return nil
}

func ReadMessage(conn net.Conn) (*Message, error) {
	if conn == nil {
		return nil, errors.New("connection is nil")
	}

	var header MessageHeader
	err := binary.Read(conn, binary.BigEndian, &header)
	if err != nil {
		return nil, fmt.Errorf("error reading header: %v", err)
	}

	payload := make([]byte, header.PayloadSize)
	_, err = io.ReadFull(conn, payload)
	if err != nil {
		return nil, fmt.Errorf("error reading payload: %v", err)
	}

	return &Message{Header: header, Payload: payload}, nil
}

func HandleConnection(conn net.Conn, receiveCh chan<- *Message) {
	defer conn.Close()

	for {
		msg, err := ReadMessage(conn)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Error reading message: %v\n", err)
			}
			return
		}

		receiveCh <- msg
	}
}

func SendToken(conn net.Conn, senderID int) error {
	return SendMessage(conn, senderID, TOKEN, nil)
}

func SendMarker(conn net.Conn, senderID int, snapshotID int) error {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(snapshotID))
	return SendMessage(conn, senderID, MARKER, payload)
}

func ParseMarkerPayload(payload []byte) (int, error) {
	if len(payload) != 4 {
		return 0, errors.New("invalid marker payload size")
	}
	return int(binary.BigEndian.Uint32(payload)), nil
}
