package main

import (
	"errors"
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

type Method string

const (
	GET  Method = "GET"
	POST Method = "POST"
	// PUT, DELETE
)

var ErrInvalidMethod = errors.New("invalid or unsupported HTTP method")

func parseMethod(s string) (Method, error) {
	switch strings.ToUpper(s) {
	case "GET":
		return GET, nil
	case "POST":
		return POST, nil
	default:
		return "", ErrInvalidMethod
	}
}

type Request struct {
	reqLine RequestLine
	headers Headers
	body    string
}

type RequestLine struct {
	method  Method
	target  string
	version string
}

type Headers map[string]string

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

func handleConn(conn net.Conn) {
	req, err := readRequest(conn)
	if err != nil {
		return
	}

	writeBuf := createResponse(req)

	sendResponse(conn, writeBuf)
}

func readRequest(conn net.Conn) (Request, error) {
	readBuf := make([]byte, maxReadBytes)
	n, err := conn.Read(readBuf)
	if err != nil {
		fmt.Println("Error reading connection: ", err.Error())
		return Request{}, err
	}
	readBuf = readBuf[:n]
	reqStr := string(readBuf)
	reqLineStr, headersBodyStr, _ := strings.Cut(reqStr, "\r\n")
	reqLine, err := getReqLine(reqLineStr)
	if err != nil {
		return Request{}, err
	}
	headersStr, body, _ := strings.Cut(headersBodyStr, "\r\n\r\n")
	headers := getHeaders(headersStr)
	return Request{
		reqLine: reqLine,
		headers: headers,
		body:    body,
	}, nil
}

func getReqLine(reqLine string) (RequestLine, error) {
	reqLineParts := strings.Fields(reqLine)
	if len(reqLineParts) != 3 {
		return RequestLine{}, fmt.Errorf("invalid request line")
	}
	method, err := parseMethod(reqLineParts[0])
	if err != nil {
		return RequestLine{}, err
	}
	return RequestLine{
		method:  method,
		target:  reqLineParts[1],
		version: reqLineParts[2],
	}, nil
}

func createResponse(req Request) (buf []byte) {
	switch req.reqLine.method {
	case GET:
		if !strings.HasPrefix(req.reqLine.target, "/") {
			buf = fmt.Appendf(nil, "%s\r\n\r\n", statusNotFound)
		}
		pathParts := strings.Split(req.reqLine.target, "/")[1:]
		if len(pathParts) >= 2 && pathParts[0] == "echo" {
			buf = createStrLenPlainTxtResp(pathParts[1])
		} else if pathParts[0] == "user-agent" {
			buf = createStrLenPlainTxtResp(req.headers["User-Agent"])
		} else if len(pathParts) >= 2 && pathParts[0] == "files" {
			buf = readFileIntoBuf(pathParts[1])
		} else {
			buf = fmt.Appendf(nil, "%s\r\n\r\n", statusOK)
		}
	case POST:
	}
	return buf
}

func sendResponse(conn net.Conn, buf []byte) {
	_, err := conn.Write(buf)
	if err != nil {
		fmt.Println("Error sending a response: ", err.Error())
		os.Exit(1)
	}
}

func getHeaders(headersStr string) Headers {
	headers := strings.Split(headersStr, "\r\n")
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
