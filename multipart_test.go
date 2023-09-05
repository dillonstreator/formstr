package formstr

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateMultipartFormFilesReader(t *testing.T) {
	assert := assert.New(t)

	file1Bytes := make([]byte, 100)
	_, err := rand.Read(file1Bytes)
	assert.NoError(err)

	file2Bytes := make([]byte, 250)
	_, err = rand.Read(file2Bytes)
	assert.NoError(err)

	body, contentType := CreateMultipartForm(context.Background(), []MultipartFormValueEncoder{
		&MultipartFormFile{
			Fieldname: "file1",
			Filename:  "file1.txt",
			Reader:    bytes.NewReader(file1Bytes),
		},
		&MultipartFormFile{
			Fieldname: "file2",
			Filename:  "file2.txt",
			Reader:    bytes.NewReader(file2Bytes),
		},
	})
	assert.Contains(contentType, "multipart/form-data; boundary=")

	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		f1, _, err := r.FormFile("file1")
		assert.NoError(err)

		f1b, err := io.ReadAll(f1)
		assert.NoError(err)

		assert.Equal(file1Bytes, f1b)

		f2, _, err := r.FormFile("file2")
		assert.NoError(err)

		f2b, err := io.ReadAll(f2)
		assert.NoError(err)

		assert.Equal(file2Bytes, f2b)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL, body)
	assert.NoError(err)

	req.Header.Add("Content-Type", contentType)

	res, err := server.Client().Do(req)
	assert.NoError(err)

	assert.Equal(http.StatusOK, res.StatusCode)
	assert.True(called)
}

type errorReader struct {
	err error
}

func (er *errorReader) Read(p []byte) (int, error) {
	return 0, er.err
}

func TestCreateMultipartFormFilesReader_error(t *testing.T) {
	assert := assert.New(t)

	file1Bytes := make([]byte, 100)
	_, err := rand.Read(file1Bytes)
	assert.NoError(err)

	errReading := errors.New("reading")

	body, contentType := CreateMultipartForm(context.Background(), []MultipartFormValueEncoder{
		&MultipartFormFile{
			Fieldname: "file1",
			Filename:  "file1.txt",
			Reader:    bytes.NewReader(file1Bytes),
		},
		&MultipartFormFile{
			Fieldname: "file2",
			Filename:  "file2.txt",
			Reader:    &errorReader{errReading},
		},
	})
	assert.Contains(contentType, "multipart/form-data; boundary=")

	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		_, _, err := r.FormFile("file1")
		assert.ErrorIs(err, io.ErrUnexpectedEOF)
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL, body)
	assert.NoError(err)

	req.Header.Add("Content-Type", contentType)

	res, err := server.Client().Do(req)
	assert.ErrorIs(err, errReading)
	assert.Nil(res)

	assert.True(called)
}
