package main

import (
	"fmt"
	"net"
	"os"
)

func main() {

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	buf := []byte("HTTP/1.1 200 OK\r\n\r\n")
	_, err = conn.Write(buf)
	if err != nil {
		fmt.Println("Error sending a response: ", err.Error())
		os.Exit(1)
	}
}
