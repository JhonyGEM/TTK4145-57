package network

import (
	"bufio"
	"config"
	"encoding/json"
	"log"
	"net"
	"strings"
	"time"
)

type Client struct {
    Conn   *net.TCPConn
    Reader *bufio.Reader
	Writer *bufio.Writer
    Addr   string
	Stop   chan struct{}
}

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

		client := New_client(conn)
		newChan<- client
		go client.Listen(msgChan, lossChan)
		go client.Heart_beat()
	}
}

func New_client(conn *net.TCPConn) *Client {
    return &Client{
        Conn:   conn,
        Reader: bufio.NewReader(conn),
		Writer: bufio.NewWriter(conn),
        Addr:   conn.RemoteAddr().String(),
		Stop:   make(chan struct{}),
    }
}

func (c *Client) Listen(msgChan chan<- Message, lossChan chan<- *Client) {
	defer c.terminate(lossChan)

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

		message.Address = c.Addr
		msgChan<- message
	}
}

func (c *Client) terminate(lossChan chan<- *Client) {
	lossChan<- c
	c.Conn.Close()
	close(c.Stop)
}

func (c *Client) Send(header HeaderType, data *DataPayload) {
	message, _ := json.Marshal(Message{Header: header, Payload: data})

	_, err := c.Writer.WriteString(string(message) + "\n")
	if err != nil {
		log.Printf("Failed to send message: %v", err)
	}

	c.Writer.Flush()
}

func (c *Client) Heart_beat() {
	ticker := time.NewTicker(config.Heartbeat_delay)
	defer ticker.Stop()

	for {
		select {
		case <- ticker.C:
			c.Send(Heartbeat, nil)
		case <- c.Stop:
			return
		}
	}
}

func Connect(server_addr string) (*net.TCPConn, error) {
	dialer := net.Dialer{Timeout: config.Dialer_timeout}

	conn, err := dialer.Dial("tcp", server_addr)
	if err != nil {
		return nil, err
	}
	return conn.(*net.TCPConn), nil
}
