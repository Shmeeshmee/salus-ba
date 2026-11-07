package filetype

import (
	"errors"
	"testing"
)

func TestDetect_TheAcceptedTypesFromTheirBytes(t *testing.T) {
	tests := []struct {
		name string
		want Type
		head []byte
	}{
		{"pdf", PDF, pdf()},
		{"pdf after leading bytes", PDF, cat(make([]byte, 1019), pdf())},
		{"jpeg", JPEG, jpegBytes(t)},
		{"png", PNG, pngBytes(t)},
		{"mp4 isom", MP4, mp4(true, trak("avc1"))},
		{"mp4 mp42", MP4, cat(ftyp("mp42", "mp42", "isom"), mdat(10))},
		{"mp4 iso5", MP4, cat(ftyp("iso5", "iso5", "dash"), mdat(10))},
		{"mp4 avc1", MP4, cat(ftyp("avc1"), mdat(10))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Detect(tt.head)
			if err != nil || got != tt.want {
				t.Fatalf("Detect = %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestDetect_TheTypeIsTheContentType(t *testing.T) {
	for typ, want := range map[Type]string{
		PDF: "application/pdf", JPEG: "image/jpeg", PNG: "image/png", MP4: "video/mp4",
	} {
		if string(typ) != want {
			t.Errorf("%q, want %q", typ, want)
		}
	}
}

func TestDetect_RefusesEverythingElse(t *testing.T) {
	tests := []struct {
		name string
		head []byte
	}{
		{"empty", nil},
		{"text called .jpg", []byte("These bytes are plain text, whatever the file name says.\n")},
		{"pdf marker past the first 1024 bytes", cat(make([]byte, 1020), pdf())},
		{"truncated png signature", pngBytes(t)[:7]},
		{"truncated jpeg signature", jpegBytes(t)[:2]},
		{"heic", cat(ftyp("heic", "mif1", "heic"), mdat(10))},
		{"heif", cat(ftyp("mif1", "mif1", "heic"), mdat(10))},
		{"avif", cat(ftyp("avif", "avif", "mif1", "miaf"), mdat(10))},
		{"mov", cat(ftyp("qt  ", "qt  "), mdat(10))},
		{"3gp", cat(ftyp("3gp4", "isom"), mdat(10))},
		{"webm", []byte{0x1A, 0x45, 0xDF, 0xA3, 0x9F, 0x42, 0x86, 0x81, 0x01, 0x42, 0xF7, 0x81, 0x01, 'w', 'e', 'b', 'm'}},
		{"ftyp without a brand", box("ftyp")},
		{"ftyp header cut", []byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'i', 's'}},
		{"ftyp size below its brand", cat([]byte{0, 0, 0, 8, 'f', 't', 'y', 'p'}, []byte("isom0000"))},
		{"gif", []byte("GIF89a\x01\x00\x01\x00")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Detect(tt.head)
			if !errors.Is(err, ErrUnsupported) {
				t.Fatalf("Detect = %q, %v; want ErrUnsupported", got, err)
			}
			if got != "" {
				t.Fatalf("Detect returned the type %q with the error", got)
			}
		})
	}
}

func TestDetect_ARenamedFileIsWhatItsBytesSay(t *testing.T) {
	// A PNG uploaded as "scan.pdf" is a PNG: Detect never sees the name, only the bytes.
	if got, _ := Detect(pngBytes(t)); got != PNG {
		t.Fatalf("Detect = %q, want PNG", got)
	}
}

func FuzzDetect(f *testing.F) {
	f.Add(pdf())
	f.Add(pngBytes(f))
	f.Add(jpegBytes(f))
	f.Add(mp4(true, trak("avc1", "mp4a")))
	f.Add(cat(ftyp("qt  "), mdat(4)))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, head []byte) {
		typ, err := Detect(head)
		if (err == nil) == (typ == "") {
			t.Fatalf("Detect = %q, %v: exactly one of a type and an error", typ, err)
		}
		if err != nil && !errors.Is(err, ErrUnsupported) {
			t.Fatalf("unexpected error %v", err)
		}
	})
}
