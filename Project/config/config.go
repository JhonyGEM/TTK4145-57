package config

import (
	"time"
)

// Elevator
const N_floors = 4
const N_buttons = 3
const Open_duration = 3 * time.Second

// Network
const UDP_port = ":60000"
const TCP_port = ":20000"
const Broadcast_delay = 1 * time.Second
const Search_timeout = 5 * time.Second
const Dialer_timeout = 5 * time.Second
const Inactivity_timeout = 10 * time.Second
const Heartbeat_delay = 5 * time.Second
const Reconnect_delay = 5 * time.Second
const Pending_resend_delay = 60 * time.Second
const Max_retries = 1

// Master
const N_elevators = 3
const Busy_penalty = 5
const Request_timeout = N_floors * 4 * time.Second