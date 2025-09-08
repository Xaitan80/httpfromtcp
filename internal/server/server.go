package server

import (
    "fmt"
    "net"
    "sync/atomic"

    "github.com/xaitan80/httpfromtcp/internal/headers"
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
        // On parse error, return 400 with plain text error via response.Writer
        rw := response.NewWriter(conn)
        _ = rw.WriteStatusLine(response.StatusBadRequest)
        hdrs := response.GetDefaultHeaders(len(err.Error()) + 1)
        _ = rw.WriteHeaders(hdrs)
        _, _ = rw.WriteBody([]byte(err.Error() + "\n"))
        return
    }

    rw := response.NewWriter(conn)
    if s.h != nil {
        if herr := s.h(r, rw); herr != nil {
            // If handler returned an error and hasn't written anything, default error output
            if !rw.WroteAnything() {
                _ = writeHandlerError(rw, herr)
            }
            return
        }
    }
    // If handler didn't write anything, write default empty 200
    if !rw.WroteAnything() {
        _ = rw.WriteStatusLine(response.StatusOK)
        hdrs := response.GetDefaultHeaders(0)
        _ = rw.WriteHeaders(hdrs)
        // no body
    }
}

// Handler is the function signature used to handle requests.
type Handler func(r *request.Request, w *response.Writer) *HandlerError

// HandlerError represents an error returned from a Handler.
type HandlerError struct {
    Status  response.StatusCode
    Headers headers.Headers
    Body    []byte
}

// writeHandlerError writes a standardized error response.
func writeHandlerError(w *response.Writer, he *HandlerError) error {
    if he == nil {
        return nil
    }
    // Write status line
    if err := w.WriteStatusLine(he.Status); err != nil {
        return err
    }
    body := he.Body
    hdrs := he.Headers
    if hdrs == nil {
        hdrs = response.GetDefaultHeaders(len(body))
    } else {
        hdrs.Set("Content-Length", fmt.Sprintf("%d", len(body)))
        if _, ok := hdrs["Connection"]; !ok {
            hdrs.Set("Connection", "close")
        }
        if _, ok := hdrs["Content-Type"]; !ok {
            hdrs.Set("Content-Type", "text/plain")
        }
    }
    if err := w.WriteHeaders(hdrs); err != nil {
        return err
    }
    _, err := w.WriteBody(body)
    if err != nil {
        return err
    }
    return nil
}
