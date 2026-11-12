package binary

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"image"
	"runtime"
	"testing"
)

// craftedJPEG is a synthetic progressive JPEG of w×h pixels and comps components (1 gray, 3 YCbCr, 4 CMYK), each
// sampled 1×1, as an attacker crafts one: a single DC-first scan of zero differences (a flat image), a few hundred
// kilobytes for 40 megapixels.
func craftedJPEG(w, h, comps int) []byte {
	var b bytes.Buffer
	b.Write([]byte{0xFF, 0xD8, 0xFF, 0xDB, 0x00, 0x43, 0x00})
	b.Write(bytes.Repeat([]byte{1}, 64))
	if comps == 4 { // Adobe's APP14, without a transform: CMYK
		b.Write([]byte{0xFF, 0xEE, 0x00, 0x0E, 'A', 'd', 'o', 'b', 'e', 0x00, 0x64, 0, 0, 0, 0, 0})
	}
	n := 8 + 3*comps
	b.Write([]byte{0xFF, 0xC2, byte(n >> 8), byte(n), 8, byte(h >> 8), byte(h), byte(w >> 8), byte(w), byte(comps)})
	for i := range comps {
		b.Write([]byte{byte(i + 1), 0x11, 0})
	}
	// One DC code, of one bit, for a difference of category 0.
	b.Write([]byte{0xFF, 0xC4, 0x00, 0x14, 0x00, 1})
	b.Write(make([]byte, 15))
	b.WriteByte(0)
	sl := 6 + 2*comps
	b.Write([]byte{0xFF, 0xDA, byte(sl >> 8), byte(sl), byte(comps)})
	for i := range comps {
		b.Write([]byte{byte(i + 1), 0x00})
	}
	b.Write([]byte{0, 0, 0})
	blocks := ((w + 7) / 8) * ((h + 7) / 8) * comps
	b.Write(make([]byte, (blocks+7)/8))
	b.Write([]byte{0xFF, 0xD9})
	return b.Bytes()
}

// adam7 are the PNG interlace passes: their first column and row, and their steps.
var adam7 = [7][4]int{{0, 0, 8, 8}, {4, 0, 8, 8}, {0, 4, 4, 8}, {2, 0, 4, 4}, {0, 2, 2, 4}, {1, 0, 2, 2}, {0, 1, 1, 2}}

// craftedPNG is a synthetic 16-bit RGB PNG of w×h black pixels, interlaced (Adam7) or not, its rows streamed into
// zlib so that making one of 40 megapixels takes little memory.
func craftedPNG(t testing.TB, w, h int, interlaced bool) []byte {
	t.Helper()
	var idat bytes.Buffer
	z, err := zlib.NewWriterLevel(&idat, zlib.BestSpeed)
	if err != nil {
		t.Fatal(err)
	}
	passes := [][4]int{{0, 0, 1, 1}}
	if interlaced {
		passes = adam7[:]
	}
	for _, p := range passes {
		pw, ph := (w-p[0]+p[2]-1)/p[2], (h-p[1]+p[3]-1)/p[3]
		if pw <= 0 || ph <= 0 {
			continue
		}
		row := make([]byte, 1+pw*6)
		for range ph {
			if _, err := z.Write(row); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	b.Write([]byte("\x89PNG\r\n\x1a\n"))
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], uint32(w)) //nolint:gosec // a test's size
	binary.BigEndian.PutUint32(ihdr[4:], uint32(h)) //nolint:gosec // a test's size
	ihdr[8], ihdr[9] = 16, 2
	if interlaced {
		ihdr[12] = 1
	}
	for _, c := range []struct {
		typ  string
		data []byte
	}{{"IHDR", ihdr}, {"IDAT", idat.Bytes()}, {"IEND", nil}} {
		_ = binary.Write(&b, binary.BigEndian, uint32(len(c.data))) //nolint:gosec // a test's chunk
		b.WriteString(c.typ)
		b.Write(c.data)
		_ = binary.Write(&b, binary.BigEndian, crc32.ChecksumIEEE(append([]byte(c.typ), c.data...)))
	}
	return b.Bytes()
}

// allocated is how many bytes fn allocates: the peak of a decode, whose coefficients and pixels all live until it
// returns.
func allocated(fn func()) uint64 {
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	fn()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// BenchmarkDecodeCrafted measures what the standard library's decoders allocate for crafted images of 40 megapixels,
// the thumbnail's old bound (SAL-30 review, finding 1; docs/plans/chat.PROGRESS.md): bytes a pixel, by format.
func BenchmarkDecodeCrafted(b *testing.B) {
	const w, h = 8000, 5000
	for _, c := range []struct {
		name string
		file func() []byte
	}{
		{"progressive JPEG gray", func() []byte { return craftedJPEG(w, h, 1) }},
		{"progressive JPEG YCbCr 4:4:4", func() []byte { return craftedJPEG(w, h, 3) }},
		{"progressive JPEG CMYK", func() []byte { return craftedJPEG(w, h, 4) }},
		{"PNG RGB 16-bit", func() []byte { return craftedPNG(b, w, h, false) }},
		{"PNG RGB 16-bit interlaced", func() []byte { return craftedPNG(b, w, h, true) }},
	} {
		b.Run(c.name, func(b *testing.B) {
			file := c.file()
			var n uint64
			for b.Loop() {
				n = allocated(func() {
					if _, _, err := image.Decode(bytes.NewReader(file)); err != nil {
						b.Fatal(err)
					}
				})
			}
			b.ReportMetric(float64(len(file)), "file-B")
			b.ReportMetric(float64(n)/(1<<20), "decode-MiB")
			b.ReportMetric(float64(n)/(w*h), "B/px")
		})
	}
}
