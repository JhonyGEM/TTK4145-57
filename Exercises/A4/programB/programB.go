package programB

import (
	config "A4/config"
	Connect "A4/connectivity"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"time"
)

const tag = "[Program B]: "

func StartProgramB(readPort int) {

	// Start UDP before program A
	conn := Connect.Connect_UDP(config.PORT, config.IP)
	if conn == nil {
		fmt.Println(tag + "Error initializing UDP")
		return
	}

	defer conn.Close()

	CreateNewProgram(0)

	fmt.Println(tag + "Program A launched without errors...")
	fmt.Println(tag + "Receiving data from Program A...")

	last := 0

	for {
		data, err := Connect.Receive_udp(conn, 10*time.Second)
		if err != nil {
			fmt.Println(tag + "Program A is dead")
			CreateNewProgram(last)
		}

		last_string := string(data)
		last, err = strconv.Atoi(last_string)

		if err != nil {
			fmt.Println(tag+"Error converting to integer", err)
			continue
		}

		fmt.Println(tag + "Data from Program A is: " + last_string)
	}
}

func CreateNewProgram(value int) error {
	// Open a new window in Windows
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		if value == 0 {
			cmd = exec.Command("cmd", "/C", "start", "powershell", "-NoExit", "go", "run", "programA/programA.go")
		} else {
			cmd = exec.Command("cmd", "/C", "start", "powershell", "-NoExit", "go", "run", "programA/programA.go", fmt.Sprint(value))
		}
	case "linux":
		if value == 0 {
			cmd = exec.Command("gnome-terminal", "--", "go", "run", "programA/programA.go")
		} else {
			cmd = exec.Command("gnome-terminal", "--", "go", "run", "programA/programA.go", fmt.Sprint(value))
		}
	}
	err := cmd.Start()

	if err != nil {
		fmt.Println(tag+"An unexpected error occurred when launching the new window", err)
		return err
	}

	return nil

}
