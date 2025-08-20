package request

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

func (r *RequestLine) ValidHTTP() bool {
	return r.HttpVersion == "HTTP/1.1"
}

type Request struct {
	RequestLine RequestLine
	Headers     map[string]string
	Body        []byte
}

var ErrMalformedRequestLine = fmt.Errorf("malformed request line")
var ErrUnsupportedHTTPVersion = fmt.Errorf("unsupported HTTP version")
var ErrEmptyRequest = fmt.Errorf("empty or incomplete request")
var SEPARATOR = "\r\n"

func parseRequestLine(b string) (*RequestLine, string, error) {
	if b == "" {
		return nil, "", ErrEmptyRequest
	}
	idx := strings.Index(b, SEPARATOR)
	if idx == -1 {
		return nil, b, ErrEmptyRequest
	}
	startLine := b[:idx]
	restOfMsg := b[idx+len(SEPARATOR):]
	parts := strings.Split(startLine, " ")
	if len(parts) != 3 {
		return nil, restOfMsg, ErrMalformedRequestLine
	}
	httpParts := strings.Split(parts[2], "/")
	if len(httpParts) != 2 || httpParts[0] != "HTTP" {
		return nil, restOfMsg, ErrMalformedRequestLine
	}
	if httpParts[1] != "1.1" {
		return nil, restOfMsg, ErrUnsupportedHTTPVersion
	}
	rl := &RequestLine{
		Method:        parts[0],
		RequestTarget: parts[1],
		HttpVersion:   httpParts[1],
	}
	return rl, restOfMsg, nil
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	bufReader := bufio.NewReader(reader)
	var requestData strings.Builder

	line, err := bufReader.ReadString('\n')
	if err != nil {
		if err == io.EOF || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, ErrEmptyRequest
		}
		return nil, errors.Join(fmt.Errorf("unable to read request line"), err)
	}
	requestData.WriteString(line)
	log.Printf("Raw request line: %q", line)

	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return nil, ErrEmptyRequest
	}

	rl, _, err := parseRequestLine(line + "\r\n")
	if err != nil {
		log.Printf("parseRequestLine error: %v", err)
		return nil, err
	}

	headers := make(map[string]string)
	for {
		line, err := bufReader.ReadString('\n')
		if err != nil {
			if err == io.EOF || errors.Is(err, io.ErrUnexpectedEOF) {
				log.Printf("EOF during headers, proceeding with partial request")
				break
			}
			log.Printf("Header read error: %v", err)
			return nil, errors.Join(fmt.Errorf("unable to read headers"), err)
		}
		requestData.WriteString(line)
		log.Printf("Raw header line: %q", line)
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			headers[parts[0]] = parts[1]
		}
	}

	var body []byte
	if contentLength, ok := headers["Content-Length"]; ok {
		length, err := strconv.Atoi(contentLength)
		if err != nil {
			log.Printf("Invalid Content-Length: %v", err)
			return nil, fmt.Errorf("invalid Content-Length: %v", err)
		}
		if length > 0 {
			body = make([]byte, length)
			_, err = io.ReadFull(bufReader, body)
			if err != nil && err != io.EOF && !errors.Is(err, io.ErrUnexpectedEOF) {
				log.Printf("Body read error: %v", err)
				return nil, errors.Join(fmt.Errorf("unable to read body"), err)
			}
			log.Printf("Raw body: %q", string(body))
		}
	}

	return &Request{
		RequestLine: *rl,
		Headers:     headers,
		Body:        body,
	}, nil
}
