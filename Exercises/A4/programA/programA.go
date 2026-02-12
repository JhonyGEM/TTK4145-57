package main

import (
	config "A4/config"
	UDP "A4/connectivity"
	"fmt"
	"os"
	"strconv"
	"time"
)

const tag = "[Program A]: "

func StartProgramA(value int) {
	fmt.Println(tag + "Starting Program A")
	// Set the starting time of the program
	start := time.Now()

	timeToLive, err := time.ParseDuration(config.TTL)

	if err != nil {
		fmt.Println(tag+"Error with the initialization of the time", err)
		return
	}

	count := value
	// for loop until time is archived
	for time.Since(start) < timeToLive {

		// Counter for testing the UDP
		count = count + 1
		sendUDP(fmt.Sprint(count))
		fmt.Println(tag+"Sent: ", count)

		// Some delay
		time.Sleep(5 * time.Second)
	}

	fmt.Printf("%sRan %s seconds\n", tag, config.TTL)
	fmt.Println(tag + "Finishing Program A")
}

func sendUDP(msg string) {
	UDP.Send_udp(msg, config.IP, config.PORT)
}

// Execute the program as go.file number
func main() {
	// If no number given start with 0
	if len(os.Args) < 2 {
		StartProgramA(0)
	}

	num, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println(tag+"Error handeling the args values", err)
		return
	}

	StartProgramA(num)
}
