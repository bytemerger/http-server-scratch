package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

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
	var requestBytes = make([]byte, 1024)
	_, err := conn.Read(requestBytes)
	if err != nil {
		fmt.Println("error reading bytes", err.Error())
	}
	requestString := string(requestBytes)
	requestPath := getRequestPath(requestString)
	if requestPath != "/" {
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		conn.Close()

	} else {
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
		conn.Close()
	}
}

func getRequestPath(str string) string {
	requestLine := strings.Split(str, "\r\n")[0]
	// split by /n to get the request target
	requestLineArray := strings.Split(requestLine, " ")
	return requestLineArray[1]
}
