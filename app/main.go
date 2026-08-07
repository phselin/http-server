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

func getHeaders(req string) map[string]string {
	headers := strings.Split(req, "\r\n")
	headers = headers[1 : len(headers)-2]
	headerMap := make(map[string]string)
	for _, header := range headers {
		headerArr := strings.Split(header, ":")
		headerMap[headerArr[0]] = strings.TrimSpace(headerArr[1])
	}
	return headerMap
}

func createStrLenPlainTxtResp(s string) []byte {
	return fmt.Appendf(nil, "HTTP/1.1 200 OK\r\n"+"Content-Type: text/plain\r\n"+"Content-Length: %d\r\n\r\n%s", len(s), s)
}

func sendResponse(conn net.Conn, buf []byte) {
	_, err := conn.Write(buf)
	if err != nil {
		fmt.Println("Error sending a response: ", err.Error())
		os.Exit(1)
	}
}

const port = 4221
const maxReadBytes = 1024

func main() {

	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		fmt.Printf("Failed to bind to port %d\n", port)
		os.Exit(1)
	}

	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	buf := make([]byte, maxReadBytes)
	n, err := conn.Read(buf)
	buf = buf[:n]
	req := string(buf)
	reqTarget := getReqTarget(req)
	headers := getHeaders(req)
	if reqTarget != "/" {
		buf = []byte("HTTP/1.1 404 Not Found\r\n\r\n")
	}

	pathParts := strings.Split(reqTarget, "/")[1:]
	if len(pathParts) >= 2 && pathParts[0] == "echo" {
		s := pathParts[1]
		buf = createStrLenPlainTxtResp(s)
	} else if pathParts[0] == "user-agent" {
		buf = createStrLenPlainTxtResp(headers["User-Agent"])
	} else {
		buf = []byte("HTTP/1.1 200 OK\r\n\r\n")
	}

	sendResponse(conn, buf)
}
