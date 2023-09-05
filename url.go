package formstr

import (
	"context"
	"encoding/base64"
	"io"
	"net/url"
	"sort"
	"strconv"
)

type FormURLValueEncoder interface {
	Encode(ctx context.Context, w io.Writer) error
}

// FormURLEncoder encodes entries (string, int, and io.Reader) into application/x-www-form-urlencoded Content-Type
type FormURLEncoder struct {
	entries map[string][]FormURLValueEncoder
}

// NewFormURLEncoder creates a FormURLEncoder with the default buffer size
func NewFormURLEncoder() *FormURLEncoder {
	return &FormURLEncoder{
		entries: make(map[string][]FormURLValueEncoder),
	}
}

func (f *FormURLEncoder) Add(key string, value FormURLValueEncoder) {
	f.entries[key] = append(f.entries[key], value)
}

// Encode encodes the contents of FormURLEncoder entries into w
func (fue *FormURLEncoder) Encode(ctx context.Context, w io.Writer) error {
	keys := make([]string, 0, len(fue.entries))
	for k := range fue.entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	isFirstEntry := true
	for _, key := range keys {
		for _, val := range fue.entries[key] {
			err := func() error {
				if isFirstEntry {
					isFirstEntry = false
				} else {
					_, err := w.Write([]byte("&"))
					if err != nil {
						return err
					}
				}

				keyEscaped := url.QueryEscape(key)
				_, err := w.Write([]byte(keyEscaped))
				if err != nil {
					return err
				}

				_, err = w.Write([]byte("="))
				if err != nil {
					return err
				}

				err = val.Encode(ctx, w)
				if err != nil {
					return err
				}

				return nil
			}()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (fue *FormURLEncoder) EncodeR(ctx context.Context) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		err := fue.Encode(ctx, pw)
		//nolint
		pw.CloseWithError(err)
	}()

	return pr
}

type FormURLValueReader struct {
	r io.Reader
}

func NewFormURLValueReader(r io.Reader) *FormURLValueReader {
	return &FormURLValueReader{r}
}

func (f *FormURLValueReader) Encode(ctx context.Context, w io.Writer) error {
	encoder := base64.NewEncoder(base64.StdEncoding, newURLQueryEscapeWriter(w))

	_, err := io.Copy(encoder, newContextReader(ctx, f.r))
	if err != nil {
		return err
	}

	err = encoder.Close()
	if err != nil {
		return err
	}

	return nil
}

type FormURLValueString struct {
	s string
}

func NewFormURLValueString(s string) *FormURLValueString {
	return &FormURLValueString{s}
}

func (f *FormURLValueString) Encode(ctx context.Context, w io.Writer) error {
	_, err := w.Write([]byte(url.QueryEscape(f.s)))

	return err
}

type FormURLValueInt struct {
	i int
}

func NewFormURLValueInt(i int) *FormURLValueInt {
	return &FormURLValueInt{i}
}

func (f *FormURLValueInt) Encode(ctx context.Context, w io.Writer) error {
	_, err := w.Write([]byte(strconv.Itoa(f.i)))

	return err
}

type FormURLValueBool struct {
	b bool
}

func NewFormURLValueBool(b bool) *FormURLValueBool {
	return &FormURLValueBool{b}
}

func (f *FormURLValueBool) Encode(ctx context.Context, w io.Writer) error {
	_, err := w.Write([]byte(strconv.FormatBool(f.b)))

	return err
}

type urlQueryEscapeWriter struct {
	w io.Writer
}

func newURLQueryEscapeWriter(w io.Writer) *urlQueryEscapeWriter {
	return &urlQueryEscapeWriter{w}
}

func (w *urlQueryEscapeWriter) Write(p []byte) (int, error) {
	escaped := url.QueryEscape(string(p))

	return w.w.Write([]byte(escaped))
}
