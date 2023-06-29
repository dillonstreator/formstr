package formstr

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormURLEncoder_Encode(t *testing.T) {
	assert := assert.New(t)

	docBytes := make([]byte, 50)
	_, err := rand.Read(docBytes)
	assert.NoError(err)

	fue := NewFormURLEncoder()
	fue.AddReader("doc", bytes.NewReader(docBytes))
	fue.AddInt("count", 10)
	fue.AddString("str!", "hello world!")

	b := bytes.NewBuffer(nil)
	err = fue.Encode(b)
	assert.NoError(err)
	assert.Equal(fmt.Sprintf("count=10&doc=%s&%s=%s", url.QueryEscape(string(docBytes)), url.QueryEscape("str!"), url.QueryEscape("hello world!")), b.String())
}

func TestFormURLEncoder_Encode_error(t *testing.T) {
	assert := assert.New(t)

	errReading := errors.New("reading")

	fue := NewFormURLEncoder()
	fue.AddReader("a", strings.NewReader("123"))
	fue.AddReader("z", strings.NewReader("789"))
	fue.AddReader("b", &errorReader{errReading})

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
