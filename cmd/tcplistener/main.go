package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
)

// getLinesChannel reads from f in 8-byte chunks, assembles complete lines,
// and sends each line (without newline) on a channel. It closes the file and
// the channel when done.
func getLinesChannel(f io.ReadCloser) <-chan string {
	ch := make(chan string)
	go func() {
		defer f.Close()
		defer close(ch)

		buf := make([]byte, 8)
		current := make([]byte, 0, 128)

		for {
			n, err := f.Read(buf)
			if n > 0 {
				data := buf[:n]
				for {
					if len(data) == 0 {
						break
					}
					if i := bytes.IndexByte(data, '\n'); i >= 0 {
						line := append(current, data[:i]...)
						// Trim a trailing '\r' to handle CRLF (\r\n)
						line = bytes.TrimSuffix(line, []byte{'\r'})
						// send the assembled line
						ch <- string(line)
						current = current[:0]
						data = data[i+1:]
						continue
					}
					// accumulate partial line
					current = append(current, data...)
					break
				}
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				// report read error to stderr and stop
				os.Stderr.Write([]byte("error: " + err.Error() + "\n"))
				return
			}
		}

		// flush any remaining partial line
		if len(current) > 0 {
			// Trim a trailing '\r' if present
			current = bytes.TrimSuffix(current, []byte{'\r'})
			ch <- string(current)
		}
	}()
	return ch
}

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
		fmt.Println("accepted connection")

		// Handle each connection concurrently so we keep accepting others.
		go func(c net.Conn) {
			defer func() {
				fmt.Println("closed connection")
			}()
			for line := range getLinesChannel(c) {
				// Print lines exactly, no extra formatting.
				fmt.Println(line)
			}
		}(conn)
	}
}
