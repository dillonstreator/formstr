package formstr

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type slowReader struct {
	r               io.Reader
	maxBytesPerRead int64
	timeoutPerRead  time.Duration
}

var _ io.Reader = (*slowReader)(nil)

func (sr *slowReader) Read(p []byte) (int, error) {
	if int64(len(p)) > sr.maxBytesPerRead {
		p = p[0:sr.maxBytesPerRead]
	}

	time.Sleep(sr.timeoutPerRead)
	n, err := sr.r.Read(p)
	return n, err
}

func TestFormURLEncoder_Encode(t *testing.T) {
	assert := assert.New(t)

	docBytes := make([]byte, 50)
	_, err := rand.Read(docBytes)
	assert.NoError(err)

	fue := NewFormURLEncoder()
	fue.AddReader("doc", bytes.NewReader(docBytes))
	fue.AddInt("count", 10)
	fue.AddString("str!", "hello world!")
	fue.AddBool("ok", true)

	b := bytes.NewBuffer(nil)
	err = fue.Encode(context.Background(), b)
	assert.NoError(err)
	assert.Equal(fmt.Sprintf("count=10&doc=%s&ok=true&%s=%s", url.QueryEscape(base64.StdEncoding.EncodeToString(docBytes)), url.QueryEscape("str!"), url.QueryEscape("hello world!")), b.String())
}

func TestFormURLEncoder_Encode_context_cancel_propagation(t *testing.T) {
	assert := assert.New(t)

	docBytes := make([]byte, 50)
	_, err := rand.Read(docBytes)
	assert.NoError(err)

	fue := NewFormURLEncoder()
	fue.AddReader("doc", &slowReader{bytes.NewReader(docBytes), 1, time.Millisecond * 10})
	fue.AddInt("count", 10)
	fue.AddString("str!", "hello world!")

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	cancel()

	b := bytes.NewBuffer(nil)
	err = fue.Encode(ctx, b)
	assert.ErrorIs(err, context.Canceled)
	assert.Equal("count=10&doc=", b.String())
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

		assert.Equal("a=MTIz&b=", string(b))
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL, fue.EncodeR(context.Background()))
	assert.NoError(err)

	res, err := server.Client().Do(req)
	assert.ErrorIs(err, errReading)
	assert.Nil(res)
	assert.True(called)
}
