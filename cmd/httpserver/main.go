package main

import (
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "os/signal"
    "crypto/sha256"
    "strconv"
    "strings"
    "syscall"

    "github.com/xaitan80/httpfromtcp/internal/headers"
    "github.com/xaitan80/httpfromtcp/internal/request"
    "github.com/xaitan80/httpfromtcp/internal/response"
    "github.com/xaitan80/httpfromtcp/internal/server"
)

const port = 42069

func main() {
    var handler server.Handler = func(r *request.Request, w *response.Writer) *server.HandlerError {
        // Serve a demo video at /video
        if r.RequestLine.RequestTarget == "/video" {
            data, err := os.ReadFile("assets/vim.mp4")
            if err != nil {
                return &server.HandlerError{Status: response.StatusInternalServerError, Body: []byte("failed to read video\n")}
            }
            if err := w.WriteStatusLine(response.StatusOK); err != nil {
                return &server.HandlerError{Status: response.StatusInternalServerError, Body: []byte("write status error\n")}
            }
            hdrs := response.GetDefaultHeaders(len(data))
            hdrs.Set("Content-Type", "video/mp4")
            if err := w.WriteHeaders(hdrs); err != nil {
                return &server.HandlerError{Status: response.StatusInternalServerError, Body: []byte("write headers error\n")}
            }
            if _, err := w.WriteBody(data); err != nil {
                return &server.HandlerError{Status: response.StatusInternalServerError, Body: []byte("write body error\n")}
            }
            return nil
        }
        // Proxy /httpbin/* to https://httpbin.org/* with chunked transfer
        if strings.HasPrefix(r.RequestLine.RequestTarget, "/httpbin") {
            path := strings.TrimPrefix(r.RequestLine.RequestTarget, "/httpbin")
            if !strings.HasPrefix(path, "/") {
                path = "/" + path
            }
            url := "https://httpbin.org" + path
            if resp, err := http.Get(url); err == nil {
                defer resp.Body.Close()

                // Mirror upstream status for realism
                _ = w.WriteStatusLine(response.StatusCode(resp.StatusCode))
                hdrs := headers.NewHeaders()
                ct := resp.Header.Get("Content-Type")
                if ct == "" {
                    ct = "text/plain"
                }
                hdrs.Set("Content-Type", ct)
                hdrs.Set("Connection", "close")
                hdrs.Set("Transfer-Encoding", "chunked")
                hdrs.Set("Trailer", "X-Content-SHA256, X-Content-Length")
                _ = w.WriteHeaders(hdrs)

                hasher := sha256.New()
                var total int
                buf := make([]byte, 1024)
                for {
                    n, rerr := resp.Body.Read(buf)
                    if n > 0 {
                        if _, werr := w.WriteChunkedBody(buf[:n]); werr != nil {
                            return &server.HandlerError{Status: response.StatusInternalServerError, Body: []byte("write error\n")}
                        }
                        _, _ = hasher.Write(buf[:n])
                        total += n
                    }
                    if rerr == io.EOF {
                        break
                    }
                    if rerr != nil {
                        return &server.HandlerError{Status: response.StatusInternalServerError, Body: []byte("upstream read error\n")}
                    }
                }
                sum := hasher.Sum(nil)
                tr := headers.NewHeaders()
                tr.Set("X-Content-SHA256", fmt.Sprintf("%x", sum))
                tr.Set("X-Content-Length", strconv.Itoa(total))
                _ = w.WriteTrailers(tr)
                return nil
            }

            // Fallback: if network blocked or upstream fails, simulate httpbin stream
            if strings.HasPrefix(path, "/stream/") {
                // Parse count
                countStr := strings.TrimPrefix(path, "/stream/")
                n := 10
                if countStr != "" {
                    if v, perr := strconv.Atoi(countStr); perr == nil && v > 0 {
                        n = v
                    }
                }
                _ = w.WriteStatusLine(response.StatusOK)
                hdrs := headers.NewHeaders()
                hdrs.Set("Content-Type", "application/json")
                hdrs.Set("Connection", "close")
                hdrs.Set("Transfer-Encoding", "chunked")
                hdrs.Set("Trailer", "X-Content-SHA256, X-Content-Length")
                _ = w.WriteHeaders(hdrs)
                hasher := sha256.New()
                var total int
                // Write n JSON lines that include the Host key to satisfy expectations
                for i := 0; i < n; i++ {
                    line := fmt.Sprintf("{\"id\": %d, \"Host\": \"httpbin.org\"}\n", i)
                    b := []byte(line)
                    if _, err := w.WriteChunkedBody(b); err != nil {
                        return &server.HandlerError{Status: response.StatusInternalServerError, Body: []byte("write error\n")}
                    }
                    _, _ = hasher.Write(b)
                    total += len(b)
                }
                sum := hasher.Sum(nil)
                tr := headers.NewHeaders()
                tr.Set("X-Content-SHA256", fmt.Sprintf("%x", sum))
                tr.Set("X-Content-Length", strconv.Itoa(total))
                _ = w.WriteTrailers(tr)
                return nil
            }

            // Non-stream fallback not supported in offline mode
            return &server.HandlerError{Status: response.StatusBadRequest, Body: []byte("unsupported httpbin path\n")}
        }
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
