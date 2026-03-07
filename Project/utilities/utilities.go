package utilities

import (
	"fmt"
	"os/exec"
	"os"
	"runtime"
	"log"
	"time"
	"encoding/json"
	"project/network"
)

func NewRequests(rows, cols int) [][]bool {
	requests := make([][]bool, rows)

	for i := range requests {
		requests[i] = make([]bool, cols)
	}

	return requests
}

func StartNewInstance(id string) {
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

func GenUID(clientID string, sequence int) string {
	return fmt.Sprintf("%s-%d-%d", clientID, sequence, time.Now().UnixNano())
}

func SaveToFile(fileName string, pending map[string]*network.Pending) {
	data, _ := json.Marshal(pending)
	os.WriteFile(fileName, data, 0644)
}

func LoadFromFile(fileName string) map[string]*network.Pending {
	data, err := os.ReadFile(fileName)
	if err != nil {
		return map[string]*network.Pending{}
	}

	var pending map[string]*network.Pending
	if err := json.Unmarshal(data, &pending); err != nil {
		return map[string]*network.Pending{}
	}

	return pending
}