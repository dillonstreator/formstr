package formstr

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
)

type MultipartFormValueEncoder interface {
	Encode(ctx context.Context, mw *multipart.Writer) error
}

func CreateMultipartForm(ctx context.Context, values []MultipartFormValueEncoder) (io.Reader, string) {
	pr, pw := io.Pipe()

	mw := multipart.NewWriter(pw)

	go func() {
		var err error
		defer func() {
			// Conditionally close the multipart writer so that
			// in the event of an error, the contents of the reader are no longer
			// valid multipart/form-data as it will not contain the closing boundary.
			// This will ensure that both the caller and callee fail processing together.
			// https://www.rfc-editor.org/rfc/rfc2046#section-5.1
			var closeErr error
			if err == nil {
				closeErr = mw.Close()
			}
			//nolint
			pw.CloseWithError(errors.Join(err, closeErr))
		}()

		for _, value := range values {
			err = value.Encode(ctx, mw)
			if err != nil {
				return
			}
		}
	}()

	return pr, mw.FormDataContentType()
}

type MultipartFormFile struct {
	Fieldname string
	Filename  string
	Reader    io.Reader
}

var _ MultipartFormValueEncoder = (*MultipartFormFile)(nil)

func (mff *MultipartFormFile) Encode(ctx context.Context, mw *multipart.Writer) error {
	w, err := mw.CreateFormFile(mff.Fieldname, mff.Filename)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, newContextReader(ctx, mff.Reader))
	if err != nil {
		return err
	}

	return nil
}

type MultipartFormField struct {
	Fieldname string
	Value     string
}

var _ MultipartFormValueEncoder = (*MultipartFormField)(nil)

func (mff *MultipartFormField) Encode(ctx context.Context, mw *multipart.Writer) error {
	return mw.WriteField(mff.Fieldname, mff.Value)
}
