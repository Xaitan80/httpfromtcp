package request

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// chunkReader simulates a reader that returns a fixed number of bytes per Read call.
type chunkReader struct {
	data            string
	numBytesPerRead int
	pos             int
}

// Read reads up to len(p) or numBytesPerRead bytes per call from the underlying string.
func (cr *chunkReader) Read(p []byte) (n int, err error) {
	if cr.pos >= len(cr.data) {
		return 0, io.EOF
	}
	endIndex := cr.pos + cr.numBytesPerRead
	if endIndex > len(cr.data) {
		endIndex = len(cr.data)
	}
	n = copy(p, cr.data[cr.pos:endIndex])
	cr.pos += n
	return n, nil
}

func Test_Good_Request_Line_Chunked(t *testing.T) {
	reader := &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 3,
	}
	r, err := RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.RequestLine.Method)
	assert.Equal(t, "/", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)
}

func Test_Good_Request_Line_With_Path_Chunked(t *testing.T) {
	reader := &chunkReader{
		data:            "GET /coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 1,
	}
	r, err := RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.RequestLine.Method)
	assert.Equal(t, "/coffee", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)
}

func Test_Good_Request_Line_MaxChunk(t *testing.T) {
    data := "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n"
    reader := &chunkReader{
        data:            data,
        numBytesPerRead: len(data),
    }
    r, err := RequestFromReader(reader)
    require.NoError(t, err)
    require.NotNil(t, r)
    assert.Equal(t, "GET", r.RequestLine.Method)
    assert.Equal(t, "/", r.RequestLine.RequestTarget)
    assert.Equal(t, "1.1", r.RequestLine.HttpVersion)
}

func Test_Good_Request_Line(t *testing.T) {
	r, err := RequestFromReader(strings.NewReader("GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n"))
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.RequestLine.Method)
	assert.Equal(t, "/", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)
}

func Test_Good_Request_Line_With_Path(t *testing.T) {
	r, err := RequestFromReader(strings.NewReader("GET /coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n"))
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.RequestLine.Method)
	assert.Equal(t, "/coffee", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)
}

func Test_Good_POST_Request_With_Path(t *testing.T) {
	r, err := RequestFromReader(strings.NewReader("POST /submit HTTP/1.1\r\nHost: localhost:42069\r\nContent-Length: 0\r\n\r\n"))
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "POST", r.RequestLine.Method)
	assert.Equal(t, "/submit", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)
}

func Test_Invalid_Number_Of_Parts_In_Request_Line(t *testing.T) {
	_, err := RequestFromReader(strings.NewReader("/coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n"))
	require.Error(t, err)
}

func Test_Invalid_Method_Request_Line(t *testing.T) {
	// lower-case method should fail
	_, err := RequestFromReader(strings.NewReader("get / HTTP/1.1\r\nHost: localhost:42069\r\n\r\n"))
	require.Error(t, err)
}

func Test_Invalid_Version_Request_Line(t *testing.T) {
	_, err := RequestFromReader(strings.NewReader("GET / HTTP/2.0\r\nHost: localhost:42069\r\n\r\n"))
	require.Error(t, err)
	_, err = RequestFromReader(strings.NewReader("GET / HTTP/1.0\r\nHost: localhost:42069\r\n\r\n"))
	require.Error(t, err)
	_, err = RequestFromReader(strings.NewReader("GET / HTTX/1.1\r\nHost: localhost:42069\r\n\r\n"))
	require.Error(t, err)
}
