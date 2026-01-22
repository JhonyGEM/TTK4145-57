Pseudocode
==========

UDP   
---

UDP uses datagrams, so receiveFrom will return whenever it receives anything. The buffer size is just the maximum size of the message, it doesn't have to be "filled". This example is for broadcasting.

### Receiver



TCP
---

For TCP sockets, you may find that a call to recv() will block until the entire buffer has been filled. Either accept fixed-size messages of size 1024 (which is what the server sends), or find some functionality that avoids this.

A handy diagram describing [Berkeley Sockets](http://en.wikipedia.org/wiki/Berkeley_sockets) on Wikipedia

### Client
```C
addr = new InternetAddress(serverIP, serverPort) 
sock = new Socket(tcp) // TCP, aka SOCK_STREAM
sock.connect(addr)
// use sock.recv() and sock.send(), just like with UDP
```

### Server
```C
// Send a message to the server:  "Connect to: " <your IP> ":" <your port> "\0"

// do not need IP, because we will set it to listening state
addr = new InternetAddress(localPort)
acceptSock = new Socket(tcp)

// You may not be able to use the same port twice when you restart the program, unless you set this option
acceptSock.setOption(REUSEADDR, true)
acceptSock.bind(addr)

loop {
    // backlog = Max number of pending connections waiting to connect()
    newSock = acceptSock.listen(backlog)

    // Spawn new thread to handle recv()/send() on newSock
}
```
   

    
Shutting down sockets
=====================
Use SocketOption REUSEADDRESS, so you can use the same address when the program restarts. This way you can afford to be lazy, and not use the proper shutdown()/close() calls.


Non-blocking sockets and select()
=================================
### Aka avoiding the use of a new thread for each connection

[From the Python Sockets HowTo](http://docs.python.org/2/howto/sockets.html#non-blocking-sockets), but the concept is the same in any language.
