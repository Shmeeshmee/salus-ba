// Package filetype tells what a file is from its bytes, never from its name or a declared content type:
// Detect for the first bytes, Checker for the codecs of an MP4 as it streams, and LimitReader for a body
// that must be exactly as long as declared. It knows the four types the product accepts and nothing about
// the limits per type, which are product rules.
package filetype

import (
	"bytes"
	"encoding/binary"
	"errors"
)

// Type is an accepted file type; its value is the content type.
type Type string

// The accepted types.
const (
	PDF  Type = "application/pdf"
	JPEG Type = "image/jpeg"
	PNG  Type = "image/png"
	MP4  Type = "video/mp4"
)

var (
	// ErrUnsupported is a file of a type, or an MP4 with a codec, that is not accepted.
	ErrUnsupported = errors.New("filetype: unsupported type")
	// ErrMalformed is an MP4 whose boxes do not add up, or whose moov is larger than the Checker buffers.
	ErrMalformed = errors.New("filetype: malformed file")
	// ErrTooLarge is a body with more bytes than declared.
	ErrTooLarge = errors.New("filetype: more bytes than declared")
	// ErrShort is a body with fewer bytes than declared.
	ErrShort = errors.New("filetype: fewer bytes than declared")
)

// pdfWindow is how far into the file the PDF header may start, as PDF readers allow.
const pdfWindow = 1024

var (
	pngSignature  = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}
	jpegSignature = []byte{0xFF, 0xD8, 0xFF}
	pdfSignature  = []byte("%PDF-")
)

// mp4Brands are the ISO base media major brands of an MP4. Others share the box format but not the
// container (qt for MOV, heic and mif1 for HEIF, avif, 3gp).
var mp4Brands = map[string]bool{
	"isom": true, "iso2": true, "iso3": true, "iso4": true, "iso5": true, "iso6": true,
	"mp41": true, "mp42": true, "avc1": true,
}

// Detect returns the type of the file whose first bytes are head (4 KiB is plenty), or ErrUnsupported.
func Detect(head []byte) (Type, error) {
	switch {
	case bytes.HasPrefix(head, pngSignature):
		return PNG, nil
	case bytes.HasPrefix(head, jpegSignature):
		return JPEG, nil
	case isMP4(head):
		return MP4, nil
	case bytes.Contains(head[:min(len(head), pdfWindow)], pdfSignature):
		return PDF, nil
	}
	return "", ErrUnsupported
}

// isMP4 reports whether head starts with an ftyp box, at least as long as its major brand, that names an
// ISO base media brand.
func isMP4(head []byte) bool {
	if len(head) < 12 || string(head[4:8]) != "ftyp" {
		return false
	}
	return binary.BigEndian.Uint32(head) >= 12 && mp4Brands[string(head[8:12])]
}
