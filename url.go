package formstr

import (
	"io"
	"net/url"
	"sort"
)

const (
	defaultFormURLEncoderBufferSize = 512
)

// FormURLEncoder encodes io.Reader entries into application/x-www-form-urlencoded Content-Type
type FormURLEncoder struct {
	bufferSize int
	entries    map[string][]io.Reader
}

// NewFormURLEncoder creates a FormURLEncoder with the default buffer size
func NewFormURLEncoder() *FormURLEncoder {
	return NewFormURLEncoderN(defaultFormURLEncoderBufferSize)
}

// NewFormURLEncoderN creates a FormURLEncoder with a user provided buffer size
func NewFormURLEncoderN(bufferSize int) *FormURLEncoder {
	return &FormURLEncoder{
		bufferSize: bufferSize,
		entries:    make(map[string][]io.Reader),
	}
}

func (fue *FormURLEncoder) Add(key string, value io.Reader) {
	fue.entries[key] = append(fue.entries[key], value)
}

// Encode encodes the contents of FormURLEncoder entries into w
func (fue *FormURLEncoder) Encode(w io.Writer) error {
	keys := make([]string, 0, len(fue.entries))
	for k := range fue.entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	isFirstEntry := true
	for _, key := range keys {
		for _, reader := range fue.entries[key] {
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

				pr, pw := io.Pipe()

				go func() {
					for {
						buf := make([]byte, fue.bufferSize)
						n, err := reader.Read(buf)
						if err != nil {
							//nolint
							pw.CloseWithError(err)
							return
						}

						_, err = pw.Write([]byte(url.QueryEscape(string(buf[:n]))))
						if err != nil {
							//nolint
							pw.CloseWithError(err)
							return
						}
					}
				}()

				_, err = io.Copy(w, pr)
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

func (fue *FormURLEncoder) EncodeR() io.Reader {
	pr, pw := io.Pipe()

	go func() {
		err := fue.Encode(pw)
		//nolint
		pw.CloseWithError(err)
	}()

	return pr
}
