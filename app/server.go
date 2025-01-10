package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
)

type Response struct {
	code    int
	reason  string
	headers map[string]string
	body    []byte
}

func (res *Response) compressBody(encoding string) error {
	if encoding == "gzip" {
		var b bytes.Buffer
		gz := gzip.NewWriter(&b)
		_, err := gz.Write(res.body)
		if err != nil {
			return &buildResponseError{
				message: "An error occured while compressing the body",
			}
		}
		if err = gz.Flush(); err != nil {
			return &buildResponseError{
				message: "An error occured while compressing the body",
			}
		}
		if err = gz.Close(); err != nil {
			return &buildResponseError{
				message: "An error occured while compressing the body",
			}
		}
		res.body = b.Bytes()
		return nil
	}
	return &buildResponseError{
		message: "compression not supported",
	}
}

func (res *Response) buildResponse(req *Request) string {
	allowedCompressions := []string{"gzip"}
	requestLine := fmt.Sprintf("HTTP/1.1 %d %v\r\n", res.code, res.reason)

	// check for compression "Accept-Encoding"

	if reqEncodingVals, ok := req.headers["Accept-Encoding"]; ok {
		vals := strings.Split(reqEncodingVals, ", ")
		var allowedReqEncoding []string
		for _, item := range vals {
			if slices.Contains(allowedCompressions, item) {
				allowedReqEncoding = append(allowedReqEncoding, item)
			}
		}
		// finally compress the body
		if len(allowedReqEncoding) > 0 {
			res.compressBody(allowedReqEncoding[0])
			res.headers["Content-Encoding"] = allowedReqEncoding[0]
		}
	}

	res.headers["Content-Length"] = fmt.Sprint(len(res.body))

	var headersString string
	for key, val := range res.headers {
		headersString += fmt.Sprintf("%v: %v \r\n", key, val)
	}
	return fmt.Sprintf("%v%v\r\n%v", requestLine, headersString, string(res.body))
}

func (res *Response) Ok(req *Request, code int) string {
	res.headers = map[string]string{}
	res.code = code
	res.reason = "OK"
	// reason is created for 201
	if code == 201 {
		res.reason = "Created"
	}
	return res.buildResponse(req)
}

func (res *Response) notFound(req *Request) string {
	res.headers = map[string]string{}
	res.code = 404
	res.reason = "Not Found"
	return res.buildResponse(req)
}

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

type buildResponseError struct {
	message string
}

func (e *buildResponseError) Error() string {
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
		res := (&Response{}).Ok(&request, 200)
		conn.Write([]byte(res))
	} else if strings.Contains(request.path, "echo") {
		responseBody := strings.Split(request.path[1:], "/")[1]
		res := (&Response{
			headers: map[string]string{
				"Content-Type": "text/plain",
			},
			code:   200,
			reason: "OK",
			body:   []byte(responseBody),
		}).buildResponse(&request)

		conn.Write([]byte(res))
	} else if strings.EqualFold(request.path[1:], "user-agent") {
		responseBody := request.headers["User-Agent"]
		res := (&Response{
			headers: map[string]string{
				"Content-Type": "text/plain",
			},
			code:   200,
			reason: "OK",
			body:   []byte(responseBody),
		}).buildResponse(&request)
		conn.Write([]byte(res))
	} else if strings.Contains(request.path[1:], "file") {
		fileName := strings.Split(request.path[1:], "/")[1]
		if request.method == "POST" {
			// write to the file instead
			err := os.WriteFile(fileDirectory+fileName, request.body, 0644)
			if err != nil {
				res := (&Response{
					code:   500,
					reason: "Server Error",
				}).buildResponse(&request)
				conn.Write([]byte(res))
			}
			res := (&Response{}).Ok(&request, 201)
			conn.Write([]byte(res))
			return
		}
		fileContent, err := os.ReadFile(fileDirectory + fileName)
		if err != nil {
			conn.Write([]byte((&Response{}).notFound(&request)))
		}
		res := (&Response{
			code:   200,
			reason: "OK",
			headers: map[string]string{
				"Content-Type": "application/octet-stream",
			},
			body: fileContent,
		}).buildResponse(&request)
		conn.Write([]byte(res))
	} else {
		conn.Write([]byte((&Response{}).notFound(&request)))
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
