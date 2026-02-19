package network

import (
	"config"
	"fmt"
	"log"
	"net"
	"time"
)

func broadcast() {
	addr, err := net.ResolveUDPAddr("udp", "255.255.255.255"+config.UDP_port)
	if err != nil {
		log.Fatalf("Failed to resolve address: %v", err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatalf("Failed to create connection: %v", err)
	}

	defer conn.Close()

	for {
		conn.Write([]byte("Hello"))
		time.Sleep(config.Broadcast_delay)
	}
}

func Discover_server() (string, error) {
	addr, err := net.ResolveUDPAddr("udp", config.UDP_port)
	if err != nil {
		return "", err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return "", err
	}

	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(config.Search_timeout))

	buffer := make([]byte, 1024)
	n, addr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		return "", err
	}
	log.Printf("Received broadcast from %s: %s\n", addr.IP, string(buffer[:n]))

	return fmt.Sprintf("%s"+config.TCP_port, addr.IP), nil
}
