package server

import (
    "fmt"
    "net"
    "sync/atomic"
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

    const body = "Hello World!\n"
    const headers = "HTTP/1.1 200 OK\r\n" +
        "Content-Type: text/plain\r\n" +
        "Content-Length: 13\r\n" +
        "\r\n"

    // Write headers and body.
    _, _ = conn.Write([]byte(headers))
    _, _ = conn.Write([]byte(body))
}

