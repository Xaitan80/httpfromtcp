package main

import (
    "bytes"
    "io"
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
            ch <- string(current)
        }
    }()
    return ch
}

func main() {
    f, err := os.Open("messages.txt")
    if err != nil {
        os.Stderr.Write([]byte("error: " + err.Error() + "\n"))
        os.Exit(1)
    }

    for line := range getLinesChannel(f) {
        // write: "read: " + line + "\n" in a single write
        out := make([]byte, 6+len(line)+1)
        copy(out, "read: ")
        copy(out[6:], line)
        out[len(out)-1] = '\n'
        if _, werr := os.Stdout.Write(out); werr != nil {
            os.Stderr.Write([]byte("error: " + werr.Error() + "\n"))
            os.Exit(1)
        }
    }
}
