package server

import (
    "bytes"
    "fmt"
    "net"
    "sync/atomic"

    "github.com/xaitan80/httpfromtcp/internal/request"
    "github.com/xaitan80/httpfromtcp/internal/response"
)

type Server struct {
    ln     net.Listener
    closed atomic.Bool
    h      Handler
}

// Serve starts a TCP listener on the given port and begins accepting
// connections in a background goroutine.
func Serve(port int, h Handler) (*Server, error) {
    ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        return nil, err
    }
    s := &Server{ln: ln, h: h}
    go s.listen()
    return s, nil
}

// Close stops the server and closes the underlying listener.
func (s *Server) Close() error {
    if s == nil {
        return nil
    }
    s.closed.Store(true)
    if s.ln != nil {
        return s.ln.Close()
    }
    return nil
}

// listen accepts connections until the server is closed, handling each in a goroutine.
func (s *Server) listen() {
    for {
        conn, err := s.ln.Accept()
        if err != nil {
            if s.closed.Load() {
                return
            }
            // Ignore transient errors and continue accepting
            continue
        }
        go s.handle(conn)
    }
}

// handle writes a fixed HTTP response and closes the connection.
func (s *Server) handle(conn net.Conn) {
    defer conn.Close()
    r, err := request.RequestFromReader(conn)
    if err != nil {
        // On parse error, return 400 with error message
        _ = writeHandlerError(conn, &HandlerError{Status: response.StatusBadRequest, Message: err.Error() + "\n"})
        return
    }

    // Buffer for handler response body
    var buf bytes.Buffer
    if s.h != nil {
        if herr := s.h(r, &buf); herr != nil {
            _ = writeHandlerError(conn, herr)
            return
        }
    }

    // Success: write 200 + headers + body
    _ = response.WriteStatusLine(conn, response.StatusOK)
    hdrs := response.GetDefaultHeaders(buf.Len())
    _ = response.WriteHeaders(conn, hdrs)
    if buf.Len() > 0 {
        _, _ = conn.Write(buf.Bytes())
    }
}

// Handler is the function signature used to handle requests.
type Handler func(r *request.Request, w *bytes.Buffer) *HandlerError

// HandlerError represents an error returned from a Handler.
type HandlerError struct {
    Status  response.StatusCode
    Message string
}

// writeHandlerError writes a standardized error response.
func writeHandlerError(w net.Conn, he *HandlerError) error {
    if he == nil {
        return nil
    }
    // Write status line
    if err := response.WriteStatusLine(w, he.Status); err != nil {
        return err
    }
    body := []byte(he.Message)
    hdrs := response.GetDefaultHeaders(len(body))
    if err := response.WriteHeaders(w, hdrs); err != nil {
        return err
    }
    if len(body) > 0 {
        if _, err := w.Write(body); err != nil {
            return err
        }
    }
    return nil
}
