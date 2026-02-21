package utilities

import (
	"fmt"
	"os/exec"
	"runtime"
	"log"
	"time"
)

func Create_request_arr(rows, cols int) [][]bool {
	request := make([][]bool, rows)

	for i := range request {
		request[i] = make([]bool, cols)
	}

	return request
}

func Start_new_instance(id string) {
	switch runtime.GOOS {
	case "windows":
		exec.Command("cmd", "/C", "start", "powershell", "go", "run", "main.go", fmt.Sprintf("-id=%s", id)).Run()
	case "linux":
		exec.Command("gnome-terminal", "--", "go", "run", "main.go", fmt.Sprintf("-id=%s", id)).Run()
	default:
		log.Print("Not supported os.")
	}
}

func Abs(x int) int {
	if x < 0 {
		return - x
	}
	return x
}

func Gen_uid(client_id string) string {
	return fmt.Sprintf("%s-%d", client_id, time.Now().UnixNano())
}