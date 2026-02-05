package programB

import (
	UDP "A4/connectivity"
	"fmt"
	"os/exec"
)

func StartProgramB(readPort int) int {

	// Open a new window in Windows
	cmd := exec.Command("cmd", "/K", "start", "powershell", "-NoExit", "go", "run", "programA/programA.go")

	err := cmd.Start()

	if err != nil {
		fmt.Println("An unexpected error ocurred when launching the new window", err)
		return -1
	}

	fmt.Println("[Program B]: Program A launched without errors...")
	fmt.Println("[Program B]: Receiving data from Program A...")

	udpChannel, err := UDP.Receive_udp(20000)
	for data := range udpChannel {
		fmt.Println("[Program B]: Data from Program A is: ", data)
	}
	return 1
}
