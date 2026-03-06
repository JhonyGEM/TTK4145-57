package master

import (
	"project/config"
	"project/elevio"
	"fmt"
	"strings"
)

func printSeparator() {
	cell := "+" + strings.Repeat("-", config.Cell_width + 1)
	fmt.Println(strings.Repeat(cell, config.N_floors) + "+")
}

func (m *Master) printHallRequest() {
	fmt.Println("Hall request")
	printSeparator()
	for b := elevio.ButtonType(0); b < elevio.ButtonType(2); b++ {
		for f := 0; f < config.N_floors; f++ {
			if (f == 3 && b == elevio.BT_HallUp) || (f == 0 && b == elevio.BT_HallDown) {
				fmt.Printf("| %-*s", config.Cell_width, " ")
				continue
			}
			if !m.HallRequests[f][b] {
				fmt.Printf("| %-*s", config.Cell_width, "-")
			} else if m.HallAssignments[f][b] == "" {
				fmt.Printf("| %-*s", config.Cell_width, "*")
			} else {
				fmt.Printf("| %-*s", config.Cell_width, m.HallAssignments[f][b])
			}
		}
		fmt.Println("|")
	}
	printSeparator()
}

func (m *Master) printCabRequest() {
	fmt.Println("Cab request")
	printSeparator()
	for _, client := range m.ClientList {
		for f := 0; f < config.N_floors; f++ {
			if m.CabRequests[f][client.ID] {
				fmt.Printf("| %-*s", config.Cell_width, client.ID)
			} else {
				fmt.Printf("| %-*s", config.Cell_width, "-")
			}
		}
		fmt.Println("|")
	}
	printSeparator()
}

func (m *Master) printClientList() {
	fmt.Println("Client list")
	printSeparator()
	fmt.Printf("| %-*s| %-*s| %-*s| %-*s|\n", config.Cell_width, "ID", config.Cell_width, "Floor", config.Cell_width, "Obst", config.Cell_width, "AR")
	printSeparator()
	for _, client := range m.ClientList {
		fmt.Printf("| %-*s| %-*d| %-*t| %-*d|\n", config.Cell_width, client.ID, config.Cell_width, client.CurrentFloor, config.Cell_width, client.Obstruction, config.Cell_width, client.ActiveReq)
	}
	printSeparator()
}

func (m *Master) Print() {
	m.printClientList()
	m.printHallRequest()
	m.printCabRequest()
}