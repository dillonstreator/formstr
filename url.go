package formstr

import (
	"fmt"
	"io"
	"net/url"
	"reflect"
	"sort"
	"strconv"
)

const (
	defaultFormURLEncoderBufferSize = 512
)

// FormURLEncoder encodes entries (string, int, and io.Reader) into application/x-www-form-urlencoded Content-Type
type FormURLEncoder struct {
	bufferSize int
	entries    map[string][]any
}

// NewFormURLEncoder creates a FormURLEncoder with the default buffer size
func NewFormURLEncoder() *FormURLEncoder {
	return NewFormURLEncoderN(defaultFormURLEncoderBufferSize)
}

// NewFormURLEncoderN creates a FormURLEncoder with a user provided buffer size
func NewFormURLEncoderN(bufferSize int) *FormURLEncoder {
	return &FormURLEncoder{
		bufferSize: bufferSize,
		entries:    make(map[string][]any),
	}
}

func (f *FormURLEncoder) AddString(key string, value string) {
	f.entries[key] = append(f.entries[key], value)
}

func (f *FormURLEncoder) AddInt(key string, value int) {
	f.entries[key] = append(f.entries[key], value)
}

func (f *FormURLEncoder) AddReader(key string, value io.Reader) {
	f.entries[key] = append(f.entries[key], value)
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
		for valIdx, val := range fue.entries[key] {
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

				switch v := val.(type) {
				case io.Reader:
					pr, pw := io.Pipe()

					go func() {
						for {
							buf := make([]byte, defaultFormURLEncoderBufferSize)
							n, err := v.Read(buf)
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

				case string:
					_, err = w.Write([]byte(url.QueryEscape(v)))
					if err != nil {
						return err
					}

				case int:
					_, err = w.Write([]byte(url.QueryEscape(strconv.Itoa(v))))
					if err != nil {
						return err
					}

				default:
					return fmt.Errorf("invalid form url encoder value type '%s' for key %s[%d]", reflect.TypeOf(v).String(), key, valIdx)
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
