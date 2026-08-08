package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
)

const port = 4221
const maxReadBytes = 1024
const statusOK = "HTTP/1.1 200 OK"
const statusNotFound = "HTTP/1.1 404 Not Found"

var dirPath *string

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
	return fmt.Appendf(nil, "%s\r\n"+"Content-Type: text/plain\r\n"+"Content-Length: %d\r\n\r\n%s", statusOK, len(s), s)
}

func sendResponse(conn net.Conn, buf []byte) {
	_, err := conn.Write(buf)
	if err != nil {
		fmt.Println("Error sending a response: ", err.Error())
		os.Exit(1)
	}
}

func readRequest(conn net.Conn) (reqTarget string, headers map[string]string, err error) {
	readBuf := make([]byte, maxReadBytes)
	n, err := conn.Read(readBuf)
	if err != nil {
		fmt.Println("Error reading connection: ", err.Error())
		return "", nil, err
	}
	readBuf = readBuf[:n]
	req := string(readBuf)
	reqTarget = getReqTarget(req)
	headers = getHeaders(req)
	return reqTarget, headers, nil
}

func readFileIntoBuf(filename string) []byte {
	if dirPath == nil {
		return fmt.Appendf(nil, "%s\r\n\r\n", statusNotFound)
	}
	filePath := fmt.Sprintf("%s/%s", *dirPath, filename)
	_, err := os.Stat(filePath)
	if err != nil {
		return fmt.Appendf(nil, "%s\r\n\r\n", statusNotFound)
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Appendf(nil, "%s\r\n\r\n", statusNotFound)
	}
	return fmt.Appendf(nil, "%s\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", statusOK, len(content), string(content))
}

func createResponse(reqTarget string, headers map[string]string) (buf []byte) {
	if !strings.HasPrefix(reqTarget, "/") {
		buf = fmt.Appendf(nil, "%s\r\n\r\n", statusNotFound)
	}

	pathParts := strings.Split(reqTarget, "/")[1:]
	if len(pathParts) >= 2 && pathParts[0] == "echo" {
		buf = createStrLenPlainTxtResp(pathParts[1])
	} else if pathParts[0] == "user-agent" {
		buf = createStrLenPlainTxtResp(headers["User-Agent"])
	} else if len(pathParts) >= 2 && pathParts[0] == "files" {
		buf = readFileIntoBuf(pathParts[1])
	} else {
		buf = fmt.Appendf(nil, "%s\r\n\r\n", statusOK)
	}
	return buf
}

func handleConn(conn net.Conn) {
	reqTarget, headers, err := readRequest(conn)
	if err != nil {
		return
	}

	writeBuf := createResponse(reqTarget, headers)

	sendResponse(conn, writeBuf)
}

func main() {

	dirPath = flag.String("directory", ".", "root directory for file requests")
	flag.Parse()

	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		fmt.Printf("Failed to bind to port %d\n", port)
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleConn(conn)
	}

}
