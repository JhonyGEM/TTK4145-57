package network

import (
	"bufio"
	"elevio"
	"config"
	"log"
	"net"
	"strings"
	"time"
	"encoding/json"
)

type DataPayload struct {
	ID           string                 `json:"id,omitempty"`
	CurrentFloor int                    `json:"current_floor,omitempty"`
	Obstruction  bool					`json:"obstruction,omitempty"`
	OrderFloor   int                    `json:"order_floor,omitempty"`
	OrderButton  elevio.ButtonType      `json:"order_button,omitempty"`
}

type Message struct {
	Header  string      			`json:"header"`
	Payload *DataPayload			`json:"payload,omitempty"`
}

type Client struct {
    Conn   *net.TCPConn
    Reader *bufio.Reader
	Writer *bufio.Writer
    Addr   string
	Stop   chan struct{}
}

// TODO: Change to send client instead of conn for new connections and disconnections
func Start_server(lossChan chan<- *Client, newChan chan<- *Client, msgChan chan<- Message) {
	go broadcast()

	addr, err := net.ResolveTCPAddr("tcp", config.TCP_port)
	if err != nil {
		log.Fatalf("Failed to resolve address: %v", err)
	}

	ln, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	defer ln.Close()

	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		client := new_client(conn)
		newChan<- client
		go client.Listen(msgChan, lossChan)
		go client.Heart_beat()
	}
}

func new_client(conn *net.TCPConn) *Client {
    return &Client{
        Conn:   conn,
        Reader: bufio.NewReader(conn),
		Writer: bufio.NewWriter(conn),
        Addr:   conn.RemoteAddr().String(),
		Stop:   make(chan struct{}),
    }
}

func (c *Client) Listen(msgChan chan<- Message, lossChan chan<- *Client) {
	defer c.disconnect(lossChan)

	for {
		c.Conn.SetReadDeadline(time.Now().Add(config.Inactivity_timeout))

		data, err := c.Reader.ReadString('\n')
		if err != nil {
			return
		}
		data = strings.TrimSpace(data)
		var message Message
		err = json.Unmarshal([]byte(data), &message)
		if err != nil {
			log.Printf("Failed to decode message %v", err)
			continue
		}

		log.Printf("Received from: %s ", c.Addr)
		msgChan<- message
	}
}

func (c *Client) disconnect(lossChan chan<- *Client) {
	lossChan<- c
	c.Conn.Close()
	close(c.Stop)
	//log.Printf("Client %s disconnected.", c.Addr)
}

func (c *Client) Send(header string, data *DataPayload) {
	message, _ := json.Marshal(Message{Header: header, Payload: data})

	_, err := c.Writer.WriteString(string(message) + "\n")
	if err != nil {
		log.Printf("Failed to send message: %v", err)
	}

	c.Writer.Flush()
}

func (c *Client) Heart_beat() {
	ticker := time.NewTicker(config.Heart_beat_delay)
	defer ticker.Stop()

	for {
		select {
		case <- ticker.C:
			c.Send("Beat", nil)
		case <- c.Stop:
			return
		}
	}
}

func Connect(server_addr string) (net.Conn, error) {
	dialer := net.Dialer{Timeout: config.Dialer_timeout}

	conn, err := dialer.Dial("tcp", server_addr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
