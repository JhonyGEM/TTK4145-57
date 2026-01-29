package config

import (
	"time"
)

// Elevator
const N_floors = 4
const N_buttons = 3
const Door_open_duration = 3 * time.Second

// Network
const UDP_port = ":60000"
const TCP_port = ":20000"
const Broadcast_delay = 1 * time.Second
const Search_timeout = 5 * time.Second
const Dialer_timeout = 5 * time.Second
const Inactivity_timeout = 10 * time.Second
const Heart_beat_delay = 5 * time.Second