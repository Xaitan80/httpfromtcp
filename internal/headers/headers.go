package headers

import (
    "bytes"
    "errors"
    "strings"
)

// Headers represents a simple HTTP headers map.
type Headers map[string]string

// NewHeaders creates an empty Headers map.
func NewHeaders() Headers {
    return make(Headers)
}

// Parse consumes at most one header line from data and updates the map.
// It returns n (bytes consumed), done (true iff an empty line was found), and err.
// Behavior:
// - If no CRLF is found, returns (0, false, nil) and consumes nothing.
// - If CRLF is at the start ("\r\n"), returns (2, true, nil) indicating end of headers.
// - Otherwise parses a single "key: value" line. Leading/trailing whitespace around
//   key and value is trimmed, but there must be no whitespace immediately before the colon.
func (h Headers) Parse(data []byte) (n int, done bool, err error) {
    // Find the end of the next line.
    crlf := []byte("\r\n")
    idx := bytes.Index(data, crlf)
    if idx == -1 {
        return 0, false, nil
    }
    // If the line is empty, we are done with headers.
    if idx == 0 { // starts with CRLF
        return 2, true, nil
    }

    line := data[:idx]
    // Split on the first ':' only (values can contain ':').
    colon := bytes.IndexByte(line, ':')
    if colon == -1 {
        return 0, false, errors.New("invalid header: missing colon")
    }
    // Enforce no whitespace between key and colon.
    if colon > 0 {
        prev := line[colon-1]
        if prev == ' ' || prev == '\t' {
            return 0, false, errors.New("invalid header: space before colon")
        }
    }

    rawKey := string(line[:colon])
    rawVal := string(line[colon+1:])

    key := strings.TrimSpace(rawKey)
    val := strings.TrimSpace(rawVal)
    if key == "" {
        return 0, false, errors.New("invalid header: empty key")
    }

    // Validate key characters (letters, digits, and '-').
    for i := 0; i < len(key); i++ {
        c := key[i]
        if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
            return 0, false, errors.New("invalid header: invalid character in key")
        }
    }

    // Normalize key to lowercase before storing.
    key = strings.ToLower(key)

    // Mutate the headers map with the parsed key/value.
    h[key] = val

    // Consume exactly this line and its CRLF, not beyond.
    return idx + 2, false, nil
}
