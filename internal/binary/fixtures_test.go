package binary_test

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

// The fixtures are generated: a few pixels, a minimal PDF and an MP4 skeleton whose boxes carry zeros (or a
// counter) where a real file carries media. Nothing in them comes from a real photo, document or video.

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

func u32(v uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return b
}

// trak is a track whose sample description lists the sample entries (avc1, mp4a, hvc1 …).
func trak(entries ...string) []byte {
	stsd := [][]byte{u32(0), u32(uint32(len(entries)))}
	for _, e := range entries {
		stsd = append(stsd, box(e, make([]byte, 8)))
	}
	return box("trak", box("tkhd", make([]byte, 84)),
		box("mdia", box("mdhd", make([]byte, 24)), box("hdlr", make([]byte, 25)),
			box("minf", box("stbl", box("stsd", stsd...), box("stts", make([]byte, 8))))))
}

// mp4Head is an MP4's ftyp and moov (an H.264 and an AAC track, or the codec given) and the header of an
// mdat of n bytes that follow it.
func mp4Head(n int64, video string) []byte {
	ftyp := box("ftyp", []byte("isom\x00\x00\x02\x00isomiso2avc1mp41"))
	moov := box("moov", box("mvhd", make([]byte, 100)), trak(video), trak("mp4a"))
	return bytes.Join([][]byte{ftyp, moov, u32(uint32(8 + n)), []byte("mdat")}, nil)
}

// mp4 is a whole generated MP4 of about n bytes of media.
func mp4(n int) []byte {
	b := mp4Head(int64(n), "avc1")
	return append(b, counting(int64(n))...)
}

// counting is n bytes of a repeating counter, so a misplaced range shows.
func counting(n int64) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i * 7)
	}
	return b
}

func pdf() []byte {
	return []byte("%PDF-1.4\n1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n" +
		"2 0 obj << /Type /Pages /Kids [] /Count 0 >> endobj\ntrailer << /Root 1 0 R >>\n%%EOF\n")
}

func picture() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for x := range 64 {
		img.Set(x, x, color.RGBA{R: 40, G: 200, B: byte(x * 4), A: 255})
	}
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
