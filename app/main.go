package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func getReqTarget(req string) string {
	reqLine := strings.Split(req, "\r\n")[0]
	return strings.Fields(reqLine)[1]
}

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

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	buf = buf[:n]
	req := string(buf)
	reqTarget := getReqTarget(req)
	if reqTarget != "/" {
		buf = []byte("HTTP/1.1 404 Not Found\r\n\r\n")
	} else {
		buf = []byte("HTTP/1.1 200 OK\r\n\r\n")
	}

	_, err = conn.Write(buf)
	if err != nil {
		fmt.Println("Error sending a response: ", err.Error())
		os.Exit(1)
	}
}
