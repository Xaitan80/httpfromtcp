package server

import (
    "fmt"
    "net"
    "sync/atomic"

    "github.com/xaitan80/httpfromtcp/internal/response"
)

type Server struct {
    ln     net.Listener
    closed atomic.Bool
}

// Serve starts a TCP listener on the given port and begins accepting
// connections in a background goroutine.
func Serve(port int) (*Server, error) {
    ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        return nil, err
    }
    s := &Server{ln: ln}
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

    // Default empty response with headers
    _ = response.WriteStatusLine(conn, response.StatusOK)
    hdrs := response.GetDefaultHeaders(0)
    _ = response.WriteHeaders(conn, hdrs)
}
