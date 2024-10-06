package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
)

type Message struct {
	SenderID int         `json:"SenderID"`
	Content  interface{} `json:"Content"`
}

type Marker struct {
	SnapshotID int `json:"SnapshotID"`
}

func SendMessage(conn net.Conn, message *Message) error {
	if conn == nil {
		return errors.New("connection is nil")
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshaling message: %v", err)
	}

	_, err = conn.Write(append(jsonData, '\n'))
	return err
}

func (m *Message) UnmarshalJSON(data []byte) error {
	var raw struct {
		SenderID int             `json:"SenderId"`
		Content  json.RawMessage `json:"Content"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m.SenderID = raw.SenderID

	var marker Marker
	if err := json.Unmarshal(raw.Content, &marker); err == nil {
		m.Content = &marker
		return nil
	}

	var strContent string
	if err := json.Unmarshal(raw.Content, &strContent); err == nil {
		m.Content = strContent
		return nil
	}

	m.Content = raw.Content
	return nil
}

func ReadMessage(conn net.Conn) (int, interface{}, error) {
	if conn == nil {
		return 0, nil, errors.New("connection is nil")
	}

	reader := bufio.NewReader(conn)
	jsonData, err := reader.ReadBytes('\n')
	if err != nil {
		return 0, nil, fmt.Errorf("error reading message: %v", err)
	}

	var message Message
	err = json.Unmarshal(jsonData, &message)
	if err != nil {
		return 0, nil, fmt.Errorf("error unmarshaling message: %v", err)
	}

	return message.SenderID, message.Content, nil
}

func HandleConnection(conn net.Conn, receiveCh chan<- *Message) {
	defer conn.Close()

	for {
		senderId, content, err := ReadMessage(conn)
		if err != nil {
			return
		}

		// Send the received message to the channel
		receiveCh <- &Message{
			SenderID: senderId,
			Content:  content,
		}
	}
}
