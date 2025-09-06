package request

import (
	"errors"
	"io"
	"strings"
)

type Request struct {
	RequestLine RequestLine
}

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

// RequestFromReader parses an HTTP request from reader.
// Currently it only parses the request-line and ignores the rest.
func RequestFromReader(reader io.Reader) (*Request, error) {
	b, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	s := string(b)
	// HTTP uses CRLF line endings. Take the first line only.
	i := strings.Index(s, "\r\n")
	if i == -1 {
		return nil, errors.New("malformed request: no CRLF in request line")
	}
	line := s[:i]

	rl, err := parseRequestLine(line)
	if err != nil {
		return nil, err
	}
	return &Request{RequestLine: rl}, nil
}

func parseRequestLine(line string) (RequestLine, error) {
	// Split into exactly 3 parts: METHOD SP REQUEST-TARGET SP HTTP/VERSION
	parts := strings.Fields(line)
	if len(parts) != 3 {
		return RequestLine{}, errors.New("invalid request line: want 3 parts")
	}

	method := parts[0]
	// Validate method: only A-Z
	for i := 0; i < len(method); i++ {
		c := method[i]
		if c < 'A' || c > 'Z' {
			return RequestLine{}, errors.New("invalid method")
		}
	}

	target := parts[1]

	versionPart := parts[2]
	const prefix = "HTTP/"
	if !strings.HasPrefix(versionPart, prefix) {
		return RequestLine{}, errors.New("invalid http version format")
	}
	ver := strings.TrimPrefix(versionPart, prefix)
	if ver != "1.1" {
		return RequestLine{}, errors.New("unsupported http version")
	}

	return RequestLine{
		Method:        method,
		RequestTarget: target,
		HttpVersion:   ver,
	}, nil
}
