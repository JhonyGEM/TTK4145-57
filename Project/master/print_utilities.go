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
	for button := elevio.ButtonType(0); button < elevio.ButtonType(2); button++ {
		for floor := 0; floor < config.N_floors; floor++ {
			if (floor == 3 && button == elevio.BT_HallUp) || (floor == 0 && button == elevio.BT_HallDown) {
				fmt.Printf("| %-*s", config.Cell_width, " ")
				continue
			}
			if !m.Requests.Hall[floor][button] {
				fmt.Printf("| %-*s", config.Cell_width, "-")
			} else if m.Requests.HallAssignments[floor][button] == "" {
				fmt.Printf("| %-*s", config.Cell_width, "*")
			} else {
				fmt.Printf("| %-*s", config.Cell_width, m.Requests.HallAssignments[floor][button])
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
		for floor := 0; floor < config.N_floors; floor++ {
			if m.Requests.Cab[floor][client.ID] {
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
	fmt.Printf("| %-*s| %-*s| %-*s| %-*s|\n", config.Cell_width, "ID", config.Cell_width, "Floor", config.Cell_width, "Obst", config.Cell_width, "ARC")
	printSeparator()
	for _, client := range m.ClientList {
		fmt.Printf("| %-*s| %-*d| %-*t| %-*d|\n", config.Cell_width, client.ID, config.Cell_width, client.CurrentFloor, config.Cell_width, client.Obstruction, config.Cell_width, client.ActiveRequestCount)
	}
	printSeparator()
}

func (m *Master) Print() {
	m.printClientList()
	m.printHallRequest()
	m.printCabRequest()
}