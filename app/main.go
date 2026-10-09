package main

import (
	"bytes"
	"compress/gzip"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const port = 4221
const maxReadBytes = 1024
const statusOK = "HTTP/1.1 200 OK"
const statusCreated = "HTTP/1.1 201 Created"
const statusNotFound = "HTTP/1.1 404 Not Found"
const statusBadRequest = "HTTP/1.1 400 Bad Request"
const statusInternalServerError = "HTTP/1.1 500 Internal Server Error"

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

	dirPathFlag := flag.String("directory", ".", "root directory for file requests")
	flag.Parse()
	dirPath := *dirPathFlag

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
		go handleConn(conn, dirPath)
	}
}

func handleConn(conn net.Conn, dirPath string) {
	defer conn.Close()

	req, err := readRequest(conn)
	if err != nil {
		return
	}

	writeBuf := createResponse(req, dirPath)

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
		return RequestLine{}, errors.New("invalid request line")
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

func createResponse(req Request, dirPath string) []byte {
	var buf []byte
	pathParts := strings.Split(req.reqLine.target, "/")[1:]
	scheme, compress := checkCompression(req.headers)
	encodingHeader := ""
	if compress && checkSchemeSupport(scheme, "gzip") {
		encodingHeader = "Content-Encoding: gzip\r\n"
	}
	switch req.reqLine.method {
	case GET:
		if !strings.HasPrefix(req.reqLine.target, "/") {
			return statusOnlyResponse(statusNotFound)
		}
		if len(pathParts) == 2 && pathParts[0] == "echo" {
			buf = createStrLenPlainTxtResp(pathParts[1], encodingHeader)
		} else if pathParts[0] == "user-agent" {
			buf = createStrLenPlainTxtResp(req.headers["User-Agent"], encodingHeader)
		} else if len(pathParts) == 2 && pathParts[0] == "files" {
			buf = readFileIntoBuf(pathParts[1], dirPath, encodingHeader)
		} else {
			buf = statusOnlyResponse(statusOK)
		}
	case POST:
		if len(pathParts) == 2 && pathParts[0] == "files" {
			buf = writeBufToFile(pathParts[1], req.headers["Content-Length"], req.body, dirPath)
		}
	}
	return buf
}

func checkCompression(headers Headers) (scheme string, ok bool) {
	scheme, ok = headers["Accept-Encoding"]
	return
}

func filePath(dirPath string, filename string) string {
	return filepath.Join(dirPath, filename)
}

func writeBufToFile(filename string, lenS string, content string, dirPath string) []byte {
	contentLen, err := strconv.Atoi(lenS)
	if err != nil {
		return statusOnlyResponse(statusBadRequest)
	}
	data := []byte(content[:contentLen])
	err = os.WriteFile(filePath(dirPath, filename), data, 0644)
	if err != nil {
		return statusOnlyResponse(statusInternalServerError)
	}
	return statusOnlyResponse(statusCreated)
}

func sendResponse(conn net.Conn, buf []byte) {
	_, err := conn.Write(buf)
	if err != nil {
		fmt.Println("Error sending a response: ", err.Error())
	}
}

func getHeaders(headersStr string) Headers {
	headers := strings.Split(headersStr, "\r\n")
	headerMap := make(map[string]string)
	for _, header := range headers {
		name, value, found := strings.Cut(header, ":")
		if !found {
			continue
		}
		headerMap[name] = strings.TrimSpace(value)
	}
	return headerMap
}

func statusOnlyResponse(status string) []byte {
	return fmt.Appendf(nil, "%s\r\n\r\n", status)
}

func checkSchemeSupport(schemes string, s string) bool {
	for scheme := range strings.SplitSeq(schemes, ",") {
		if strings.EqualFold(strings.TrimSpace(scheme), s) {
			return true
		}
	}
	return false
}

func createStrLenPlainTxtResp(s string, encodingHeader string) []byte {
	if encodingHeader != "" {
		comp, err := gzipCompression([]byte(s))
		if err == nil {
			s = string(comp)
		}
	}
	return fmt.Appendf(nil, "%s\r\n%sContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", statusOK, encodingHeader, len(s), s)
}

func readFileIntoBuf(filename string, dirPath string, encodingHeader string) []byte {
	path := filePath(dirPath, filename)
	_, err := os.Stat(path)
	if err != nil {
		return statusOnlyResponse(statusNotFound)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return statusOnlyResponse(statusNotFound)
	}
	if encodingHeader != "" {
		comp, err := gzipCompression(content)
		if err == nil {
			content = comp
		}
	}
	return fmt.Appendf(nil, "%s\r\n%sContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", statusOK, encodingHeader, len(content), string(content))
}

func gzipCompression(b []byte) ([]byte, error) {
	var dstBuf bytes.Buffer
	gzipWriter := gzip.NewWriter(&dstBuf)
	if _, err := gzipWriter.Write(b); err != nil {
		gzipWriter.Close()
		return nil, err
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}
	return dstBuf.Bytes(), nil
}
