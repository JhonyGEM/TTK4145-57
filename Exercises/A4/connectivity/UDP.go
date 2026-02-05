package network

import (
	"fmt"
	"net"
)

func Send_udp(msg string, ip string, port int) {

	address, err := net.ResolveUDPAddr("udp", ip+":"+fmt.Sprint(port))
	conn, err := net.Dial("udp", address.String())

	if err != nil {
		fmt.Printf("Error: ", err)
	}

	defer conn.Close()

	data := []byte(msg)
	conn.Write(data)
}

func Receive_udp(port int) (<-chan []byte, error) {

	ch := make(chan []byte)

	address, err := net.ResolveUDPAddr("udp", fmt.Sprint(port))
	conn, err := net.ListenUDP("udp", address)
	if err != nil {
		fmt.Println("Error during the UDP initialization: ", err)
		return nil, err
	}

	go func() {
		defer conn.Close()

		buffer := make([]byte, 1024)

		for {
			n, _, err := conn.ReadFromUDP(buffer)
			if err != nil {
				fmt.Println("Error during the UDP receiving: ", err)
			}
			data := make([]byte, n)
			copy(data, buffer[n:])
			ch <- data
		}
	}()
	return ch, nil
}
