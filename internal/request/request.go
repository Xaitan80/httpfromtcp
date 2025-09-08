package request

import (
    "bytes"
    "errors"
    "io"
    "strings"

    "github.com/xaitan80/httpfromtcp/internal/headers"
)

type Request struct {
    RequestLine RequestLine
    Headers     headers.Headers
    Body        []byte
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
    r := &Request{state: stateInitialized, Headers: headers.NewHeaders()}

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

        // Allow state transitions that don't require additional bytes (e.g., no body)
        if len(buf) == 0 && r.state != stateDone {
            if _, err := r.parse(nil); err != nil {
                return nil, err
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
            // Allow state transitions that don't require additional data (e.g., no body)
            if r.state != stateDone {
                if _, perr := r.parse(nil); perr != nil {
                    return nil, perr
                }
            }
            if r.state == stateDone {
                return r, nil
            }
            return nil, errors.New("incomplete request")
        }
        if err != nil {
            return nil, err
        }
    }
}

type parserState int

const (
    stateInitialized parserState = iota
    stateParsingHeaders
    stateParsingBody
    stateDone
)

// parse consumes bytes from data and updates the Request.
// It returns the number of bytes consumed from data and an error if parsing fails.
func (r *Request) parse(data []byte) (int, error) {
    if r.state == stateDone {
        return 0, nil
    }
    total := 0
    for {
        n, err := r.parseSingle(data[total:])
        if err != nil {
            return total, err
        }
        if n == 0 {
            break
        }
        total += n
        if r.state == stateDone || total == len(data) {
            break
        }
    }
    return total, nil
}

// parseSingle processes a single step depending on the current parser state.
func (r *Request) parseSingle(data []byte) (int, error) {
    switch r.state {
    case stateInitialized:
        consumed, rl, err := parseRequestLine(data)
        if err != nil {
            return 0, err
        }
        if consumed == 0 {
            return 0, nil
        }
        r.RequestLine = rl
        r.state = stateParsingHeaders
        return consumed, nil
    case stateParsingHeaders:
        n, done, err := r.Headers.Parse(data)
        if err != nil {
            return 0, err
        }
        if n == 0 && !done {
            return 0, nil
        }
        if done {
            r.state = stateParsingBody
        }
        return n, nil
    case stateParsingBody:
        // Determine desired content length from headers; if missing, we're done.
        clStr := r.Headers.Get("Content-Length")
        if clStr == "" {
            r.state = stateDone
            // Nothing to parse; report consuming all provided data to advance buffer.
            return len(data), nil
        }
        // Parse content length
        var want int
        for i := 0; i < len(clStr); i++ {
            c := clStr[i]
            if c < '0' || c > '9' {
                return 0, errors.New("invalid Content-Length")
            }
            want = want*10 + int(c-'0')
        }
        // Append all data given
        if len(data) > 0 {
            r.Body = append(r.Body, data...)
        }
        // If we've reached the desired length, we're done
        if len(r.Body) == want {
            r.state = stateDone
        }
        // If we've exceeded, error
        if len(r.Body) > want {
            return len(data), errors.New("body exceeds Content-Length")
        }
        // Report consumed bytes (entire slice provided)
        return len(data), nil
    case stateDone:
        return 0, nil
    default:
        return 0, errors.New("invalid parser state")
    }
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
