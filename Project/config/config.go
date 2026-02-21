package config

import (
	"time"
)

// Elevator
const N_floors =		4
const N_buttons =		3
const Open_duration =	3 * time.Second

// Network
const UDP_port =			":60000"
const TCP_port =			":20000"
const Search_timeout = 		5 * time.Second
const Dialer_timeout = 		5 * time.Second
const Inactivity_timeout = 	10 * time.Second
const Heartbeat_interval = 	5 * time.Second
const Broadcast_rate =		1 * time.Second
const Pending_resend_rate = 60 * time.Second
const Reconnect_delay = 	5 * time.Second
const Msg_buf_size =        (N_floors * N_buttons) * 2

// Master
const N_elevators = 		3
const Busy_penalty = 		5
const Max_retries = 		1
const Request_timeout = 	N_floors * 4 * time.Second
const Resend_rate =         30 * time.Second
