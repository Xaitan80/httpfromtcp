package main

import (
    "bytes"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/xaitan80/httpfromtcp/internal/request"
    "github.com/xaitan80/httpfromtcp/internal/response"
    "github.com/xaitan80/httpfromtcp/internal/server"
)

const port = 42069

func main() {
    var handler server.Handler = func(r *request.Request, w *bytes.Buffer) *server.HandlerError {
        switch r.RequestLine.RequestTarget {
        case "/yourproblem":
            return &server.HandlerError{Status: response.StatusBadRequest, Message: "Your problem is not my problem\n"}
        case "/myproblem":
            return &server.HandlerError{Status: response.StatusInternalServerError, Message: "Woopsie, my bad\n"}
        default:
            w.WriteString("All good, frfr\n")
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
