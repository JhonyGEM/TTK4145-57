package network

import (
	"project/config"
	"encoding/json"
	"bufio"
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
	Activity chan struct{}
	Stop   chan struct{}
}

func StartServer(lossChan chan<- *Client, newChan chan<- *Client, msgChan chan<- Message) {
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

		client := NewClient(conn)
		newChan<- client
		go client.Listen(msgChan, lossChan)
		go client.Heartbeat()
	}
}

func NewClient(conn *net.TCPConn) *Client {
    return &Client{
        Conn:     conn,
        Reader:   bufio.NewReader(conn),
		Writer:   bufio.NewWriter(conn),
        Addr:     conn.RemoteAddr().String(),
		Activity: make(chan struct{}),
		Stop:     make(chan struct{}),
    }
}

func (c *Client) GetIP() string {
	addr, _, _ := net.SplitHostPort(c.Addr)
	return addr
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

func (c *Client) Send(message Message) {
	packet, _ := json.Marshal(message)

	_, err := c.Writer.WriteString(string(packet) + "\n")
	if err != nil {
		log.Printf("Failed to send message: %v", err)
	}

	c.Writer.Flush()

	select {
	case c.Activity <- struct{}{}:
	default:
	}
}

func (c *Client) Heartbeat() {
	idleTimer := time.NewTimer(config.Heartbeat_interval)
	defer idleTimer.Stop()

	for {
		select {
		case <- idleTimer.C:
			c.Send(Message{Header: Heartbeat})
			idleTimer.Reset(config.Heartbeat_interval)

		case <- c.Activity:
			if !idleTimer.Stop() {
				select {
				case <- idleTimer.C:
				default:
				}
			}
			idleTimer.Reset(config.Heartbeat_interval)

		case <- c.Stop:
			return
		}
	}
}

func Connect(serverAddr string) (*net.TCPConn, error) {
	dialer := net.Dialer{Timeout: config.Dialer_timeout}

	conn, err := dialer.Dial("tcp", serverAddr)
	if err != nil {
		return nil, err
	}
	return conn.(*net.TCPConn), nil
}

func HasInternetConnection() bool {
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", config.Dialer_timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func GetLocalIP() string {
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", config.Dialer_timeout)
	if err != nil {
		return ""
	}
	ip, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	conn.Close()
	return ip
}