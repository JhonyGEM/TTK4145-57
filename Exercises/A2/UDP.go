
package main

import (
	"fmt"
	"net"
	"time"
)

func send_udp(msg string) {
	
	address, err := net.ResolveUDPAddr("udp", "10.100.23.11:20010") // 20000 + n (10 in our case)
	conn, err := net.Dial("udp", address.String())

	if (err != nil) {
		fmt.Printf("Error: ", err)
	}

	defer conn.Close()

	data := []byte(msg)
	conn.Write(data)

	time.Sleep(5 * time.Second)
}

func receive_udp() {
	address, _ := net.ResolveUDPAddr("udp", ":20010")
	conn, _ := net.ListenUDP("udp", address)
	//if (err) {return]

	defer conn.Close()

	buffer := make([]byte, 1024)

	for{
		n, addr, _ := conn.ReadFromUDP(buffer)
		//if (err) return

		fmt.Printf("Received: %d bytes from %s: %s \n", n, addr.String(), string(buffer[:n]))
	}
}

func main1() {
	go send_udp("Hi, from the computer")
	go receive_udp()
	select{}
}