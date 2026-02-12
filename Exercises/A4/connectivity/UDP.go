package network

import (
	"fmt"
	"net"
	"time"
)

const tag = "[UDP]: "

func Send_udp(msg string, ip string, port int) {
	address, err := net.ResolveUDPAddr("udp", ip+":"+fmt.Sprint(port))

	if err != nil {
		fmt.Println(tag+"Error: ", err)
	}

	conn, err := net.Dial("udp", address.String())

	if err != nil {
		fmt.Println(tag+"Error: ", err)
	}

	defer conn.Close()

	data := []byte(msg)
	conn.Write(data)
}

func Receive_udp(conn *net.UDPConn, timeout time.Duration) ([]byte, error) {

	buffer := make([]byte, 1024)

	conn.SetDeadline(time.Now().Add(timeout))

	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		fmt.Println(tag+"Error during the UDP receiving: ", err)
		return nil, err
	}

	return buffer[:n], nil
}

func Connect_UDP(port int, ip string) *net.UDPConn {
	address, err := net.ResolveUDPAddr("udp", fmt.Sprintf(ip+":%d", port))
	conn, err := net.ListenUDP("udp", address)
	if err != nil {
		fmt.Println(tag+"Error during the UDP initialization: ", err)
		return nil
	}

	return conn
}
