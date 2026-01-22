// Use `go run foo.go` to run your program

package main

import (
    . "fmt"
    "runtime"
    "time"
)

var i = 0

type inc struct{}
type dec struct{}
type get struct{ reply chan int }

func numberServer(req chan any) {
    for {
        select {
        case msg := <-req:
            switch m := msg.(type) {
            case inc:
                i++
            case dec:
                i--
            case get:
                m.reply <- i
                return
            }
        }
    }
}

func incrementing(req chan any, done chan struct{}) {
    //TODO: increment i 1000000 times
    for k := 0; k < 1000000; k++ {
        req <- inc{}
    }
    done <- struct{}{}
}

func decrementing(req chan any, done chan struct{}) {
    //TODO: decrement i 1000000 times
    for k := 0; k < 1000000; k++ {
        req <- dec{}
    }
    done <- struct{}{}
}

func main() {
    // What does GOMAXPROCS do? What happens if you set it to 1?
    runtime.GOMAXPROCS(2)    
	
    // TODO: Spawn both functions as goroutines
    req := make(chan any)
    done := make(chan struct{})
    reply := make(chan int)

    go numberServer(req)
    go incrementing(req, done)
    go decrementing(req, done)
	
    // We have no direct way to wait for the completion of a goroutine (without additional synchronization of some sort)
    // We will do it properly with channels soon. For now: Sleep.
    <-done
    <-done
    time.Sleep(0*time.Millisecond)
    req <- get{reply: reply}
    Println("The magic number is:", <-reply)
}
