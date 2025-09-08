package headers

import (
    "bytes"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// Test: Valid single header
func Test_Valid_Single_Header(t *testing.T) {
    headers := NewHeaders()
    // Mixed case key should be normalized to lowercase in map
    data := []byte("HoSt: localhost:42069\r\n\r\n")
    n, done, err := headers.Parse(data)
    require.NoError(t, err)
    require.NotNil(t, headers)
    assert.Equal(t, "localhost:42069", headers["host"])
    // Consume only the first CRLF-terminated line, not the trailing CRLF
    assert.Equal(t, 23, n)
    assert.False(t, done)
}

// Test: Valid single header with extra whitespace
func Test_Valid_Single_Header_With_Extra_Whitespace(t *testing.T) {
    headers := NewHeaders()
    data := []byte(" hOsT:    localhost:42069   \r\n\r\n")
    n, done, err := headers.Parse(data)
    require.NoError(t, err)
    assert.Equal(t, "localhost:42069", headers["host"])
    // n should be up to the first CRLF
    exp := bytes.Index(data, []byte("\r\n")) + 2
    assert.Equal(t, exp, n)
    assert.False(t, done)
}

// Test: Valid 2 headers with existing headers
func Test_Valid_Two_Headers_With_Existing(t *testing.T) {
    headers := NewHeaders()
    headers["existing"] = "foo"

    data := []byte("HOST: localhost:42069\r\nUser-AGENT: curl\r\n\r\n")

    // First header
    n1, done1, err1 := headers.Parse(data)
    require.NoError(t, err1)
    assert.False(t, done1)
    assert.Equal(t, "localhost:42069", headers["host"])
    assert.Equal(t, "foo", headers["existing"]) // still present

    // Second header
    n2, done2, err2 := headers.Parse(data[n1:])
    require.NoError(t, err2)
    assert.False(t, done2)
    assert.Equal(t, "curl", headers["user-agent"])

    // Following should signal done (empty line)
    n3, done3, err3 := headers.Parse(data[n1+n2:])
    require.NoError(t, err3)
    assert.True(t, done3)
    assert.Equal(t, 2, n3)
}

// Test: Valid done (empty line)
func Test_Valid_Done(t *testing.T) {
    headers := NewHeaders()
    n, done, err := headers.Parse([]byte("\r\n"))
    require.NoError(t, err)
    assert.True(t, done)
    assert.Equal(t, 2, n)
    assert.Empty(t, headers)
}

// Test: Invalid spacing header
func Test_Invalid_Spacing_Header(t *testing.T) {
    headers := NewHeaders()
    data := []byte("       Host : localhost:42069       \r\n\r\n")
    n, done, err := headers.Parse(data)
    require.Error(t, err)
    assert.Equal(t, 0, n)
    assert.False(t, done)
}

// Test: Invalid character in header key
func Test_Invalid_Character_In_Key(t *testing.T) {
    headers := NewHeaders()
    data := []byte("H©st: localhost:42069\r\n\r\n")
    n, done, err := headers.Parse(data)
    require.Error(t, err)
    assert.Equal(t, 0, n)
    assert.False(t, done)
}
