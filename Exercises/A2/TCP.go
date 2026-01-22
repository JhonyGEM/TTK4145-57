package main

import (
	"fmt"
	"net"
	//"time"
)

func send_tcp(msg string) {

	address, err := net.ResolveTCPAddr("tcp", "10.100.23.11:33546") // 20000 + n (10 in our case)
	conn, err := net.Dial("tcp", address.String())

	if err != nil {
		fmt.Printf("Error: ", err)
	}

	defer conn.Close()

	conn.Write([]byte(msg))
//	buffer := make([]byte, 1024)
//		n, _ := conn.Read(buffer)
//		fmt.Printf("Received: %d bytes: %s \n", n, buffer[:n-1])
//		time.Sleep(5 * time.Second)
//	}
	
}

func receive_tcp() {
	address, err := net.ResolveTCPAddr("tcp", ":8080")
	ln, err := net.ListenTCP("tcp", address)

	if err != nil {
		fmt.Printf("Received:", err)
		return
	}
	defer ln.Close()

	buffer := make([]byte, 1024)

	conn, err := ln.Accept()

	if err != nil {
		fmt.Printf("Received:", err)
		return
	}

	defer conn.Close()

	for {
		
		n, err := conn.Read(buffer)

		if err != nil {
			fmt.Printf("Received:", err)
			return
		}

		fmt.Printf("Received: %d bytes: %s \n", n, buffer[:n-1])
	}
}

func main() {
	send_tcp("Connect to: 10.100.23.20:8080\x00")
	go receive_tcp()
	select{}
}
