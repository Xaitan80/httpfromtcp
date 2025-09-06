package main

import (
    "bytes"
    "io"
    "os"
)

func main() {
    f, err := os.Open("messages.txt")
    if err != nil {
        os.Stderr.Write([]byte("error: " + err.Error() + "\n"))
        os.Exit(1)
    }

    buf := make([]byte, 8)
    current := make([]byte, 0, 128) // holds the current line across reads

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
                    // write: "read: " + line + "\n"
                    out := make([]byte, 6+len(line)+1)
                    copy(out, "read: ")
                    copy(out[6:], line)
                    out[len(out)-1] = '\n'
                    if _, werr := os.Stdout.Write(out); werr != nil {
                        os.Stderr.Write([]byte("error: " + werr.Error() + "\n"))
                        os.Exit(1)
                    }
                    current = current[:0] // reset for next line
                    data = data[i+1:]
                    continue
                }
                // no newline in remaining data; accumulate and continue reading
                current = append(current, data...)
                break
            }
        }
        if err == io.EOF {
            break
        }
        if err != nil {
            os.Stderr.Write([]byte("error: " + err.Error() + "\n"))
            os.Exit(1)
        }
    }

    // Flush any remaining partial line
    if len(current) > 0 {
        out := make([]byte, 6+len(current)+1)
        copy(out, "read: ")
        copy(out[6:], current)
        out[len(out)-1] = '\n'
        _, _ = os.Stdout.Write(out)
    }

    _ = f.Close()
}
