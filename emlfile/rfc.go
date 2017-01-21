package emlfile

import (
	"mime"
)

// DecodeRFC2047 decode text according by RFC2047
// if met plain text here, will return the orignal text
// =?utf-8?q?=E5=BC=A0=E5=B9=BF=E6=94=BF?=
// =?utf-8?b?5byg5bm/5pS/?=
func DecodeRFC2047(text string) string {
	dec := new(mime.WordDecoder)
	ret, err := dec.Decode(text) // plain text cause error [mime: invalid RFC 2047 encoded-word]
	if err != nil {              // some charset cause error [mime: unhandled charset "GBK"]
		return text
	}
	return ret
}

// EncodeRFC2047 use mail's rfc2047 to encode any string
// See: https://godoc.org/mime#pkg-constants
// if met encoded text, will return the orignal text unchanged.
func EncodeRFC2047(s string) string {
	return mime.QEncoding.Encode("utf-8", s)
}
