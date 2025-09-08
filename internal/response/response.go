package response

import (
    "fmt"
    "io"
    "sort"

    "github.com/xaitan80/httpfromtcp/internal/headers"
)

// StatusCode is a limited set of HTTP status codes we support.
type StatusCode int

const (
    StatusOK                  StatusCode = 200
    StatusBadRequest          StatusCode = 400
    StatusInternalServerError StatusCode = 500
)

// WriteStatusLine writes the HTTP/1.1 status line for the given status code.
func WriteStatusLine(w io.Writer, statusCode StatusCode) error {
    var reason string
    switch statusCode {
    case StatusOK:
        reason = "OK"
    case StatusBadRequest:
        reason = "Bad Request"
    case StatusInternalServerError:
        reason = "Internal Server Error"
    default:
        reason = ""
    }
    if reason == "" {
        _, err := fmt.Fprintf(w, "HTTP/1.1 %d\r\n", int(statusCode))
        return err
    }
    _, err := fmt.Fprintf(w, "HTTP/1.1 %d %s\r\n", int(statusCode), reason)
    return err
}

// GetDefaultHeaders returns the default headers for our responses.
func GetDefaultHeaders(contentLen int) headers.Headers {
    h := headers.NewHeaders()
    // Use canonical case for response header keys
    h["Content-Length"] = fmt.Sprintf("%d", contentLen)
    h["Connection"] = "close"
    h["Content-Type"] = "text/plain"
    return h
}

// WriteHeaders writes headers as "Key: Value\r\n" lines and a final CRLF.
func WriteHeaders(w io.Writer, h headers.Headers) error {
    return writeHeadersInternal(w, h)
}

// writeHeadersInternal renders headers and final CRLF.
func writeHeadersInternal(w io.Writer, h headers.Headers) error {
    // Preferred order for our default headers
    order := []string{"Content-Length", "Connection", "Content-Type"}
    written := make(map[string]struct{}, len(h))
    for _, k := range order {
        if v, ok := h[k]; ok {
            if _, err := fmt.Fprintf(w, "%s: %s\r\n", k, v); err != nil {
                return err
            }
            written[k] = struct{}{}
        }
    }
    // Write any remaining headers in sorted order
    var rest []string
    for k := range h {
        if _, ok := written[k]; !ok {
            rest = append(rest, k)
        }
    }
    sort.Strings(rest)
    for _, k := range rest {
        if _, err := fmt.Fprintf(w, "%s: %s\r\n", k, h[k]); err != nil {
            return err
        }
    }
    // End of headers
    _, err := io.WriteString(w, "\r\n")
    return err
}

// Writer enforces ordered writing of status line, headers, then body.
type Writer struct {
    w     io.Writer
    state writerState
}

type writerState int

const (
    writerStateInit writerState = iota
    writerStateStatus
    writerStateHeaders
    writerStateBody
)

// NewWriter wraps an io.Writer with ordered response writing.
func NewWriter(w io.Writer) *Writer { return &Writer{w: w, state: writerStateInit} }

// WriteStatusLine writes the HTTP status line. Must be first.
func (wr *Writer) WriteStatusLine(statusCode StatusCode) error {
    if wr.state != writerStateInit {
        return fmt.Errorf("invalid write order: status already written")
    }
    if err := WriteStatusLine(wr.w, statusCode); err != nil {
        return err
    }
    wr.state = writerStateStatus
    return nil
}

// WriteHeaders writes headers after the status line.
func (wr *Writer) WriteHeaders(h headers.Headers) error {
    if wr.state != writerStateStatus {
        return fmt.Errorf("invalid write order: headers before status or after body")
    }
    if err := writeHeadersInternal(wr.w, h); err != nil {
        return err
    }
    wr.state = writerStateHeaders
    return nil
}

// WriteBody writes response body. Must be after headers.
func (wr *Writer) WriteBody(p []byte) (int, error) {
    if wr.state != writerStateHeaders && wr.state != writerStateBody {
        return 0, fmt.Errorf("invalid write order: body before headers")
    }
    wr.state = writerStateBody
    return wr.w.Write(p)
}

// WroteAnything returns true if any part of the response has been written.
func (wr *Writer) WroteAnything() bool { return wr.state != writerStateInit }
