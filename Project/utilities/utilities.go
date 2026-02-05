package utilities

import (
	"os/exec"
	"runtime"
	"log"
)

func Create_request_arr(rows, cols int) [][]bool {
	q := make([][]bool, rows)

	for i := range q {
		q[i] = make([]bool, cols)
	}

	return q
}

func Start_new_master() {
	if runtime.GOOS == "windows" {
		exec.Command("cmd", "/C", "start", "powershell", "go", "run", "run_master.go").Run()
	} else if runtime.GOOS == "linux" {
		exec.Command("gnome-terminal", "--", "go", "run", "run_master.go").Run()
	} else {
		log.Print("Not supported os.")
	}
}

func Abs(x int) int {
	if x < 0 {
		return - x
	}
	return x
}