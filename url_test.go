package formstr

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormURLEncoder_Encode(t *testing.T) {
	assert := assert.New(t)

	fue := NewFormURLEncoder()
	fue.Add("a", strings.NewReader("123"))
	fue.Add("z", strings.NewReader("789"))
	fue.Add("b", strings.NewReader("456"))

	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		b, err := io.ReadAll(r.Body)
		assert.NoError(err)

		assert.Equal("a=123&b=456&z=789", string(b))
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL, fue.EncodeR())
	assert.NoError(err)

	res, err := server.Client().Do(req)
	assert.NoError(err)

	assert.Equal(http.StatusOK, res.StatusCode)
	assert.True(called)
}

func TestFormURLEncoder_Encode_error(t *testing.T) {
	assert := assert.New(t)

	errReading := errors.New("reading")

	fue := NewFormURLEncoder()
	fue.Add("a", strings.NewReader("123"))
	fue.Add("z", strings.NewReader("789"))
	fue.Add("b", &errorReader{errReading})

	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		b, err := io.ReadAll(r.Body)
		assert.ErrorIs(err, io.ErrUnexpectedEOF)

		assert.Equal("a=123&b=", string(b))
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL, fue.EncodeR())
	assert.NoError(err)

	res, err := server.Client().Do(req)
	assert.ErrorIs(err, errReading)
	assert.Nil(res)
	assert.True(called)
}
