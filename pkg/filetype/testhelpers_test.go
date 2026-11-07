package filetype

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

// The fixtures below are generated: a few pixels, a minimal PDF and an MP4 skeleton whose boxes carry
// zeros where a real file would carry media. Nothing in them comes from a real photo, document or video.

func box(typ string, parts ...[]byte) []byte {
	size := 8
	for _, p := range parts {
		size += len(p)
	}
	b := make([]byte, 8, size)
	binary.BigEndian.PutUint32(b, uint32(size))
	copy(b[4:], typ)
	for _, p := range parts {
		b = append(b, p...)
	}
	return b
}

// largeBox writes the 64-bit size form (size field 1, then the size in the next eight bytes).
func largeBox(typ string, payload []byte) []byte {
	b := make([]byte, 16, 16+len(payload))
	binary.BigEndian.PutUint32(b, 1)
	copy(b[4:], typ)
	binary.BigEndian.PutUint64(b[8:], uint64(16+len(payload)))
	return append(b, payload...)
}

func ftyp(major string, compatible ...string) []byte {
	payload := []byte(major + "\x00\x00\x02\x00")
	for _, c := range compatible {
		payload = append(payload, c...)
	}
	return box("ftyp", payload)
}

func u32(v uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return b
}

// trak is a track whose sample description lists the given sample entries (avc1, mp4a, hvc1, Opus …).
func trak(entries ...string) []byte {
	stsd := [][]byte{u32(0), u32(uint32(len(entries)))}
	for _, e := range entries {
		stsd = append(stsd, box(e, make([]byte, 8)))
	}
	return box("trak",
		box("tkhd", make([]byte, 84)),
		box("mdia",
			box("mdhd", make([]byte, 24)),
			box("hdlr", make([]byte, 25)),
			box("minf",
				box("stbl",
					box("stsd", stsd...),
					box("stts", make([]byte, 8)),
				),
			),
		),
	)
}

func moov(traks ...[]byte) []byte {
	return box("moov", append([][]byte{box("mvhd", make([]byte, 100))}, traks...)...)
}

func mdat(n int) []byte { return box("mdat", make([]byte, n)) }

// mp4 is a whole file: ftyp, then moov before or after the media data.
func mp4(moovFirst bool, traks ...[]byte) []byte {
	head := ftyp("isom", "isom", "iso2", "avc1", "mp41")
	if moovFirst {
		return cat(head, moov(traks...), mdat(5000))
	}
	return cat(head, box("free"), mdat(5000), moov(traks...))
}

func cat(parts ...[]byte) []byte { return bytes.Join(parts, nil) }

func pdf() []byte {
	return []byte("%PDF-1.4\n1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n" +
		"2 0 obj << /Type /Pages /Kids [] /Count 0 >> endobj\ntrailer << /Root 1 0 R >>\n%%EOF\n")
}

func picture() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(1, 1, color.RGBA{R: 40, G: 200, B: 140, A: 255})
	return img
}

func pngBytes(t testing.TB) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, picture()); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func jpegBytes(t testing.TB) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, picture(), nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
