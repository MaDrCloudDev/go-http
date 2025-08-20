package request

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type chunkReader struct {
	data            string
	numBytesPerRead int
	pos             int
}

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
	if n > cr.numBytesPerRead {
		n = cr.numBytesPerRead
		cr.pos -= n - cr.numBytesPerRead
	}
	return n, nil
}

func TestRequestLineParse(t *testing.T) {
	// Test: good GET request line
	reader := &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\n\r\n",
		numBytesPerRead: 3,
	}
	r, err := RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.RequestLine.Method)
	assert.Equal(t, "/", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)

	// Test: good GET request line with path (create new reader)
	reader2 := &chunkReader{
		data:            "GET /test HTTP/1.1\r\nHost: localhost:42069\r\n\r\n",
		numBytesPerRead: 3,
	}
	r2, err := RequestFromReader(reader2)
	require.NoError(t, err)
	require.NotNil(t, r2)
	assert.Equal(t, "GET", r2.RequestLine.Method)
	assert.Equal(t, "/test", r2.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r2.RequestLine.HttpVersion)

	// Test: invalid number of parts in request line (create new reader with malformed data)
	reader3 := &chunkReader{
		data:            "GET HTTP/1.1\r\n\r\n", // Missing request target
		numBytesPerRead: 3,
	}
	_, err = RequestFromReader(reader3)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrMalformedRequestLine)
}
