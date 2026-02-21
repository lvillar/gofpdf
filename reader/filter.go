package reader

import (
	"bytes"
	"compress/zlib"
	"encoding/ascii85"
	"encoding/hex"
	"fmt"
	"io"
)

// decodeStream applies the filter chain specified in the stream dictionary to decompress data.
func decodeStream(s Stream) ([]byte, error) {
	data := s.Data
	filter := s.Dict["Filter"]

	if filter == nil {
		return data, nil
	}

	// Filter can be a single name or an array of names
	var filters []Name
	switch f := filter.(type) {
	case Name:
		filters = []Name{f}
	case Array:
		for _, item := range f {
			n, ok := item.(Name)
			if !ok {
				return nil, fmt.Errorf("reader: filter array contains non-name: %T", item)
			}
			filters = append(filters, n)
		}
	default:
		return nil, fmt.Errorf("reader: unexpected filter type: %T", filter)
	}

	var err error
	for _, f := range filters {
		data, err = applyFilter(f, data)
		if err != nil {
			return nil, fmt.Errorf("reader: applying filter %s: %w", f, err)
		}
	}
	return data, nil
}

// applyFilter applies a single decompression filter to the data.
func applyFilter(name Name, data []byte) ([]byte, error) {
	switch name {
	case "FlateDecode":
		return flateDecode(data)
	case "ASCIIHexDecode":
		return asciiHexDecode(data)
	case "ASCII85Decode":
		return ascii85Decode(data)
	default:
		return nil, fmt.Errorf("unsupported filter: %s", name)
	}
}

// flateDecode decompresses zlib/deflate encoded data.
func flateDecode(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("zlib init: %w", err)
	}
	defer r.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return nil, fmt.Errorf("zlib decompress: %w", err)
	}
	return buf.Bytes(), nil
}

// asciiHexDecode decodes ASCII hex-encoded data (terminated by '>').
func asciiHexDecode(data []byte) ([]byte, error) {
	// Remove whitespace and trailing '>'
	var clean bytes.Buffer
	for _, b := range data {
		if b == '>' {
			break
		}
		if !isWhitespace(b) {
			clean.WriteByte(b)
		}
	}

	src := clean.Bytes()
	// Pad odd-length with trailing 0
	if len(src)%2 != 0 {
		src = append(src, '0')
	}

	dst := make([]byte, hex.DecodedLen(len(src)))
	_, err := hex.Decode(dst, src)
	if err != nil {
		return nil, fmt.Errorf("ascii hex decode: %w", err)
	}
	return dst, nil
}

// ascii85Decode decodes ASCII85-encoded data (terminated by "~>").
func ascii85Decode(data []byte) ([]byte, error) {
	// Find the end marker "~>"
	end := bytes.Index(data, []byte("~>"))
	if end >= 0 {
		data = data[:end]
	}

	decoder := ascii85.NewDecoder(bytes.NewReader(data))
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, decoder); err != nil {
		return nil, fmt.Errorf("ascii85 decode: %w", err)
	}
	return buf.Bytes(), nil
}
