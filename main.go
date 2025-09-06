package main

import (
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
    // Prebuild output buffer: "read: " + up to 8 bytes + "\n"
    out := make([]byte, 6+8+1)
    copy(out, "read: ")

    for {
        n, err := f.Read(buf)
        if n > 0 {
            copy(out[6:], buf[:n])
            out[6+n] = '\n'
            if _, werr := os.Stdout.Write(out[:6+n+1]); werr != nil {
                os.Stderr.Write([]byte("error: " + werr.Error() + "\n"))
                os.Exit(1)
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

    _ = f.Close()
}
