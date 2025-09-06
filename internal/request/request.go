package request

import (
    "bytes"
    "errors"
    "io"
    "strings"
)

type Request struct {
    RequestLine RequestLine
    state       parserState
}

type RequestLine struct {
    HttpVersion   string
    RequestTarget string
    Method        string
}

// RequestFromReader parses an HTTP request from reader incrementally.
// It reads chunks and feeds them to the Request parser until the
// request-line has been fully parsed.
func RequestFromReader(reader io.Reader) (*Request, error) {
    r := &Request{state: stateInitialized}

    // Accumulation buffer for bytes read but not yet parsed/consumed.
    buf := make([]byte, 0, 32)
    tmp := make([]byte, 8)

    for {
        if len(buf) > 0 {
            consumed, err := r.parse(buf)
            if err != nil {
                return nil, err
            }
            if consumed > 0 {
                buf = buf[consumed:]
            }
            if r.state == stateDone {
                return r, nil
            }
        }

        // Need more data
        n, err := reader.Read(tmp)
        if n > 0 {
            buf = append(buf, tmp[:n]...)
        }
        if err == io.EOF {
            // Try one last parse attempt on whatever remains.
            if len(buf) > 0 && r.state != stateDone {
                if consumed, perr := r.parse(buf); perr != nil {
                    return nil, perr
                } else if consumed > 0 {
                    buf = buf[consumed:]
                }
            }
            if r.state == stateDone {
                return r, nil
            }
            return nil, errors.New("incomplete request line")
        }
        if err != nil {
            return nil, err
        }
    }
}

type parserState int

const (
    stateInitialized parserState = iota
    stateDone
)

// parse consumes bytes from data and updates the Request.
// It returns the number of bytes consumed from data and an error if parsing fails.
func (r *Request) parse(data []byte) (int, error) {
    if r.state == stateDone {
        return 0, nil
    }
    consumed, rl, err := parseRequestLine(data)
    if err != nil {
        return 0, err
    }
    if consumed == 0 {
        return 0, nil
    }
    r.RequestLine = rl
    r.state = stateDone
    return consumed, nil
}

// parseRequestLine attempts to parse a request-line from the beginning of data.
// It returns the number of bytes consumed (including the trailing CRLF),
// the parsed RequestLine, and an error. If no CRLF is found, it returns (0, _, nil).
func parseRequestLine(data []byte) (int, RequestLine, error) {
    // Find LF; require preceding CR for CRLF
    lf := bytes.IndexByte(data, '\n')
    if lf == -1 {
        return 0, RequestLine{}, nil
    }
    if lf == 0 || data[lf-1] != '\r' {
        return 0, RequestLine{}, errors.New("invalid request line ending: expected CRLF")
    }
    line := string(data[:lf-1]) // exclude CR

    parts := strings.Fields(line)
    if len(parts) != 3 {
        return 0, RequestLine{}, errors.New("invalid request line: want 3 parts")
    }

    method := parts[0]
    for i := 0; i < len(method); i++ {
        c := method[i]
        if c < 'A' || c > 'Z' {
            return 0, RequestLine{}, errors.New("invalid method")
        }
    }

    target := parts[1]

    versionPart := parts[2]
    const prefix = "HTTP/"
    if !strings.HasPrefix(versionPart, prefix) {
        return 0, RequestLine{}, errors.New("invalid http version format")
    }
    ver := strings.TrimPrefix(versionPart, prefix)
    if ver != "1.1" {
        return 0, RequestLine{}, errors.New("unsupported http version")
    }

    rl := RequestLine{
        Method:        method,
        RequestTarget: target,
        HttpVersion:   ver,
    }
    // consumed bytes include CRLF; lf is index of LF; consumed = lf+1
    return lf + 1, rl, nil
}

