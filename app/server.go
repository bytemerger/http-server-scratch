package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type Request struct {
	method  string
	path    string
	headers map[string]string
	body    []byte
}

type requestError struct {
	message string
}

func (e *requestError) Error() string {
	return e.message
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("application started !")

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
	// check if there is arg passed
	var fileDirectory string
	if len(os.Args) > 2 {
		fileDirectory = os.Args[2]
	}
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
	} else if strings.Contains(request.path[1:], "file") {
		fileName := strings.Split(request.path[1:], "/")[1]
		if request.method == "POST" {
			// write to the file instead
			err := os.WriteFile(fileDirectory+fileName, request.body, 0644)
			if err != nil {
				conn.Write([]byte("HTTP/1.1 500 Server Error\r\n\r\n"))
			}
			conn.Write([]byte("HTTP/1.1 201 Created\r\n\r\n"))
			return
		}
		fileContent, err := os.ReadFile(fileDirectory + fileName)
		if err != nil {
			conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		}
		body := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %v\r\n\r\n%v", len(fileContent), string(fileContent))
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
	remBody := []byte(headersLineArray[len(headersLineArray)-1])

	contentLength, ok := request.headers["Content-Length"]
	if !ok {
		// then we dont know the content length return everything remaining
		request.body = remBody
		return request, nil
	}
	bodyLength, err := strconv.Atoi(contentLength)
	if err != nil {
		return Request{}, &requestError{
			message: "Content-length can not be parsed properly expecting a number",
		}
	}
	request.body = remBody[:bodyLength]
	return request, nil
}
