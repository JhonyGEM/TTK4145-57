package main

import (
	UDP "A4/connectivity"
	"fmt"
	"time"
)

func StartProgramA() {
	fmt.Println("[Program A]: Starting Program A")
	// Set the starting time of the program
	start := time.Now()
	timeToLive_string := "360s"

	timeToLive, err := time.ParseDuration(timeToLive_string)

	if err != nil {
		fmt.Printf("Error with the initialization of the time\n", err)
		return
	}

	count := 0
	// for loop until time is archived
	for time.Since(start) < timeToLive {

		// Counter for testing the UDP
		count = count + 1
		sendUDP(fmt.Sprint(count))
		fmt.Println("[Program A]: Sent: ", count)

		// Some delay
		time.Sleep(5 * time.Second)
	}

	fmt.Printf("Ran %s seconds\n", timeToLive_string)
	fmt.Println("[Program A]: Finishing Program A")
}

func sendUDP(msg string) {
	UDP.Send_udp(msg, "localhost", 20000)
}

func main() {
	StartProgramA()
}
