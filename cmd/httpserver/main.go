package main

import (
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/xaitan80/httpfromtcp/internal/headers"
    "github.com/xaitan80/httpfromtcp/internal/request"
    "github.com/xaitan80/httpfromtcp/internal/response"
    "github.com/xaitan80/httpfromtcp/internal/server"
)

const port = 42069

func main() {
    var handler server.Handler = func(r *request.Request, w *response.Writer) *server.HandlerError {
        // Prepare HTML bodies
        html400 := []byte("<html>\n  <head>\n    <title>400 Bad Request</title>\n  </head>\n  <body>\n    <h1>Bad Request</h1>\n    <p>Your request honestly kinda sucked.</p>\n  </body>\n</html>\n")
        html500 := []byte("<html>\n  <head>\n    <title>500 Internal Server Error</title>\n  </head>\n  <body>\n    <h1>Internal Server Error</h1>\n    <p>Okay, you know what? This one is on me.</p>\n  </body>\n</html>\n")
        html200 := []byte("<html>\n  <head>\n    <title>200 OK</title>\n  </head>\n  <body>\n    <h1>Success!</h1>\n    <p>Your request was an absolute banger.</p>\n  </body>\n</html>\n")

        switch r.RequestLine.RequestTarget {
        case "/yourproblem":
            hdrs := headers.NewHeaders()
            hdrs.Set("Content-Type", "text/html")
            hdrs.Set("Connection", "close")
            hdrs.Set("Content-Length", "0") // will be overwritten in writeHandlerError
            return &server.HandlerError{Status: response.StatusBadRequest, Headers: hdrs, Body: html400}
        case "/myproblem":
            hdrs := headers.NewHeaders()
            hdrs.Set("Content-Type", "text/html")
            hdrs.Set("Connection", "close")
            hdrs.Set("Content-Length", "0")
            return &server.HandlerError{Status: response.StatusInternalServerError, Headers: hdrs, Body: html500}
        default:
            // Write success directly using the response.Writer
            _ = w.WriteStatusLine(response.StatusOK)
            hdrs := response.GetDefaultHeaders(len(html200))
            hdrs.Set("Content-Type", "text/html")
            _ = w.WriteHeaders(hdrs)
            _, _ = w.WriteBody(html200)
            return nil
        }
    }

    srv, err := server.Serve(port, handler)
    if err != nil {
        log.Fatalf("Error starting server: %v", err)
    }
    defer srv.Close()
    log.Println("Server started on port", port)

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
    log.Println("Server gracefully stopped")
}
