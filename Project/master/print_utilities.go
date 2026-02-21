package master

import (
	"project/config"
	"project/elevio"
	"fmt"
	"strings"
)

func print_separator() {
	cell := "+" + strings.Repeat("-", config.Cell_width + 1)
	fmt.Println(strings.Repeat(cell, config.N_floors) + "+")
}

func (m *Master) Print_hall_request() {
	fmt.Print("Hall request \n")
	print_separator()
	for b := elevio.ButtonType(0); b < elevio.ButtonType(2); b++ {
		for f := 0; f < config.N_floors; f++ {
			if (f == 3 && b == elevio.BT_HallUp) || (f == 0 && b == elevio.BT_HallDown) {
				fmt.Printf("| %-*s", config.Cell_width, " ")
				continue
			}
			if !m.Hall_requests[f][b] {
				fmt.Printf("| %-*s", config.Cell_width, "-")
			} else if m.Hall_assignments[f][b] == "" {
				fmt.Printf("| %-*s", config.Cell_width, "*")
			} else {
				fmt.Printf("| %-*s", config.Cell_width, m.Hall_assignments[f][b])
			}
		}
		fmt.Println("|")
	}
	print_separator()
}

func (m *Master) Print_cab_request() {
	fmt.Print("Cab request \n")
	print_separator()
	for _, client := range m.Client_list {
		for f := 0; f < config.N_floors; f++ {
			if m.Cab_requests[f][client.ID] {
				fmt.Printf("| %-*s", config.Cell_width, client.ID)
			} else {
				fmt.Printf("| %-*s", config.Cell_width, "-")
			}
		}
		fmt.Println("|")
	}
	print_separator()
}

func (m *Master) Print_client_list() {
	fmt.Print("Client list \n")
	print_separator()
	fmt.Printf("| %-*s| %-*s| %-*s| %-*s|\n", config.Cell_width, "ID", config.Cell_width, "Floor", config.Cell_width, "Obst", config.Cell_width, "AR")
	print_separator()
	for _, client := range m.Client_list {
		fmt.Printf("| %-*s| %-*d| %-*t| %-*d|\n", config.Cell_width, client.ID, config.Cell_width, client.Current_floor, config.Cell_width, client.Obstruction, config.Cell_width, client.Active_req)
	}
	print_separator()
}