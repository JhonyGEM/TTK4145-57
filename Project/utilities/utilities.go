package utilities

import (
	"fmt"
	"os/exec"
	"runtime"
	"log"
	"time"
)

func Create_request_arr(rows, cols int) [][]bool {
	q := make([][]bool, rows)

	for i := range q {
		q[i] = make([]bool, cols)
	}

	return q
}

func Start_new_instance(id string) {
	switch runtime.GOOS {
	case "windows":
		exec.Command("cmd", "/C", "start", "powershell", "go", "run", "main.go", "-id=%s", id).Run()
	case "linux":
		exec.Command("gnome-terminal", "--", "go", "run", "main.go").Run()
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