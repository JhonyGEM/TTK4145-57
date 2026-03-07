package config

import (
	"time"
)

// Elevator
const N_floors =		4
const N_buttons =		3
const Open_duration =	3 * time.Second
const Elev_backup =     "elevator_backup.json"

// Network
const UDP_port =			":60000"
const TCP_port =			":20000"
const Search_timeout = 		5 * time.Second
const Dialer_timeout = 		5 * time.Second
const Inactivity_timeout = 	10 * time.Second
const Heartbeat_interval = 	5 * time.Second
const UDP_broadcast_rate =  1 * time.Second
const Pending_timeout =     60 * time.Second
const Pending_check_rate =  2 * time.Second
const Reconnect_delay = 	5 * time.Second

// Master
const N_elevators = 		3
const Cost_penalty =        2
const Max_retries = 		1
const Msg_buf_size =        (N_floors * N_buttons) * 2
const Request_timeout = 	N_floors * 4 * time.Second
const Resend_rate =         30 * time.Second
const MonitorInterval =     500 * time.Millisecond
const Pending_backup =      "pending_backup.json"

// Debugging
const Cell_width =          8
