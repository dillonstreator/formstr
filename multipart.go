package formstr

import (
	"errors"
	"io"
	"mime/multipart"
)

type MultipartFormFile struct {
	Fieldname string
	Filename  string
	Reader    io.Reader
}

func (mff *MultipartFormFile) Create(mw *multipart.Writer) error {
	w, err := mw.CreateFormFile(mff.Fieldname, mff.Filename)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, mff.Reader)
	if err != nil {
		return err
	}

	return nil
}

type MultipartFormEntry interface {
	Create(mw *multipart.Writer) error
}

func CreateMultipartFormReader(entries []MultipartFormEntry) (io.Reader, string) {
	pr, pw := io.Pipe()

	writer := multipart.NewWriter(pw)

	go func() {
		var err error
		defer func() {
			// Conditionally close the multipart writer so that
			// in the event of an error, the contents of the reader are no longer
			// valid multipart/form-data as it will not contain the closing boundary.
			// This will ensure that not that both the caller and callee fail processing together.
			// https://www.rfc-editor.org/rfc/rfc2046#section-5.1
			var closeErr error
			if err == nil {
				closeErr = writer.Close()
			}
			//nolint
			pw.CloseWithError(errors.Join(err, closeErr))
		}()

		for _, entry := range entries {
			err = entry.Create(writer)
			if err != nil {
				return
			}
		}
	}()

	return pr, writer.FormDataContentType()
}
