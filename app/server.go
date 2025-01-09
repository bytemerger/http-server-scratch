package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

type Request struct {
	method  string
	path    string
	headers map[string]string
}

type requestError struct {
	message string
}

func (e *requestError) Error() string {
	return e.message
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("server could not accept connection", err.Error())
			os.Exit(1)
		}
		go handleConnections(conn)
	}
}

func handleConnections(conn net.Conn) {
	//close connection after each request
	defer conn.Close()
	var requestBytes = make([]byte, 1024)
	_, err := conn.Read(requestBytes)
	if err != nil {
		fmt.Println("error reading bytes", err.Error())
	}
	requestString := string(requestBytes)
	request, parseErr := parseRequest(requestString)
	if parseErr != nil {
		fmt.Println("The request string is invalid", err.Error())
	}
	if request.path == "/" {
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	} else if strings.Contains(request.path, "echo") {
		responseBody := strings.Split(request.path[1:], "/")[1]
		body := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %v\r\n\r\n%v", len(responseBody), responseBody)

		conn.Write([]byte(body))
	} else if strings.EqualFold(request.path[1:], "user-agent") {
		responseBody := request.headers["User-Agent"]
		body := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %v\r\n\r\n%v", len(responseBody), responseBody)
		conn.Write([]byte(body))
	} else {
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}
}

func parseRequest(str string) (Request, error) {
	requestLineString, rest, found := strings.Cut(str, "\r\n")
	if !found {
		return Request{}, &requestError{
			message: "The request string can not be parsed",
		}
	}
	// split by /n to get the request target
	requestLineArray := strings.Split(requestLineString, " ")
	var request Request
	request.method = requestLineArray[0]
	request.path = requestLineArray[1]
	request.headers = make(map[string]string)
	headersLineArray := strings.Split(rest, "\r\n")
	for _, item := range headersLineArray[:len(headersLineArray)-1] {
		// split the headers to get the key value
		// strings.Cut should also work
		if len(item) < 1 {
			continue
		}
		headerkeyValue := strings.SplitN(item, ":", 2)
		request.headers[headerkeyValue[0]] = strings.TrimSpace(headerkeyValue[1])
	}
	return request, nil
}
