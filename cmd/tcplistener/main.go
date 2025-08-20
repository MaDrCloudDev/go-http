package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"go.http.me/internal/request"
)

func handleConnection(conn net.Conn) {
	fmt.Fprint(conn, "Enter HTTP request (e.g., GET / HTTP/1.1\r\nHost: localhost:42069\r\n\r\n): ")

	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		log.Printf("error setting read deadline: %v", err)
		conn.Close()
		return
	}
	defer conn.Close()

	req, err := request.RequestFromReader(conn)
	if err != nil {
		log.Printf("error parsing request: %v", err)
		fmt.Fprintf(conn, "HTTP/1.1 400 Bad Request\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
		return
	}
	fmt.Printf("Parsed request: Method=%s, RequestTarget=%s, HttpVersion=%s, Headers=%v, Body=%s\n",
		req.RequestLine.Method, req.RequestLine.RequestTarget, req.RequestLine.HttpVersion, req.Headers, string(req.Body))
	response := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 12\r\nConnection: close\r\n\r\nHello, World!"
	fmt.Fprint(conn, response)
}

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Fatalf("error starting listener: %v", err)
	}
	defer listener.Close()
	log.Println("Server listening on :42069")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("error accepting connection: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}

// go run ./cmd/tcplistener | tee /tmp/rawpost.http
// curl -X POST -H "Content-Type: application/json" -d '{"foo":"bar"}' http://localhost:42069
// nc -v localhost 42069
// echo -e "GET / HTTP/1.1\r\nHost: localhost:42069\r\n\r\n" | nc localhost 42069
