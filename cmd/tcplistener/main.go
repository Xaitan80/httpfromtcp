package main

import (
    "fmt"
    "net"
    "os"
    "sort"

    "github.com/xaitan80/httpfromtcp/internal/request"
)

func main() {
    ln, err := net.Listen("tcp", ":42069")
    if err != nil {
        fmt.Println("listen error:", err)
        os.Exit(1)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("accept error:", err)
			continue
		}
        // connection accepted

        // Handle each connection concurrently so we keep accepting others.
        go func(c net.Conn) {
            defer c.Close()

            r, err := request.RequestFromReader(c)
            if err != nil {
                fmt.Fprintln(os.Stderr, "parse error:", err)
                return
            }

            fmt.Println("Request line:")
            fmt.Printf("- Method: %s\n", r.RequestLine.Method)
            fmt.Printf("- Target: %s\n", r.RequestLine.RequestTarget)
            fmt.Printf("- Version: %s\n", r.RequestLine.HttpVersion)

            fmt.Println("Headers:")
            if len(r.Headers) > 0 {
                keys := make([]string, 0, len(r.Headers))
                for k := range r.Headers {
                    keys = append(keys, k)
                }
                sort.Strings(keys)
                for _, k := range keys {
                    fmt.Printf("- %s: %s\n", k, r.Headers[k])
                }
            }
        }(conn)
    }
}
