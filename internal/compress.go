package internal

// compress.go — serialises the payload to JSON and compresses it with zlib,
// returning a base64-encoded string ready for chunking.

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
)

// Compress marshals the payload to JSON, compresses it with zlib, and returns
// the result as a standard base64-encoded string.
func Compress(payload *Payload) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	// RawURLEncoding uses '-' and '_' instead of '+' and '/', and omits '=' padding.
	// This makes the output safe to embed in a URL query string without percent-encoding.
	return base64.RawURLEncoding.EncodeToString(buf.Bytes()), nil
}

