package binary

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
	"time"
)

// BenchmarkThumbnail is the worker's cost of a chat photo's thumbnail (SAL-30): a synthetic 12-megapixel JPEG, as a
// phone sends one, decoded, scaled to ThumbnailEdge and encoded (the store's put aside). docs/plans/chat.PROGRESS.md.
func BenchmarkThumbnail(b *testing.B) {
	src := image.NewYCbCr(image.Rect(0, 0, 4000, 3000), image.YCbCrSubsampleRatio420)
	for i := range src.Y {
		src.Y[i] = byte(i * 7)
	}
	var photo bytes.Buffer
	if err := jpeg.Encode(&photo, src, &jpeg.Options{Quality: 90}); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.SetBytes(int64(photo.Len()))
	for b.Loop() {
		img, ok := decodeImage(photo.Bytes())
		if !ok {
			b.Fatal("the photo does not decode")
		}
		var out bytes.Buffer
		if err := jpeg.Encode(&out, scaled(img, ThumbnailEdge), &jpeg.Options{Quality: thumbnailQuality}); err != nil {
			b.Fatal(err)
		}
	}
}

// decodeImage refuses an image past maxThumbnailPixels from its header alone, before decoding a pixel.
func TestDecodeImage_RefusesAnImageOverTheBoundFromItsHeader(t *testing.T) {
	t.Parallel()
	var small, wide bytes.Buffer
	if err := jpeg.Encode(&small, image.NewGray(image.Rect(0, 0, 10, 10)), nil); err != nil {
		t.Fatal(err)
	}
	img := image.NewGray(image.Rect(0, 0, 8000, 5001)) // 40 008 000 pixels
	img.Set(0, 0, color.White)
	if err := jpeg.Encode(&wide, img, nil); err != nil {
		t.Fatal(err)
	}
	if _, ok := decodeImage(small.Bytes()); !ok {
		t.Error("a small JPEG: refused")
	}
	if _, ok := decodeImage(wide.Bytes()); ok {
		t.Error("a JPEG over 40 megapixels: decoded")
	}
	if _, ok := decodeImage([]byte("%PDF-1.4")); ok {
		t.Error("a PDF: decoded as an image")
	}
}

// decodeImage refuses, from its header alone, an image whose decode would take more than maxDecodeBytes, whatever
// its pixel count: a crafted progressive JPEG or interlaced 16-bit PNG of 40 megapixels (about 600 MB decoded,
// BenchmarkDecodeCrafted) allocates next to nothing (SAL-30 review, finding 1).
func TestDecodeImage_RefusesACraftedImagePastTheMemoryBudgetFromItsHeader(t *testing.T) {
	for name, file := range map[string][]byte{
		"progressive JPEG of 40 MP":      craftedJPEG(8000, 5000, 3),
		"progressive JPEG of 14 MP":      craftedJPEG(4000, 3500, 3),
		"progressive CMYK JPEG of 9 MP":  craftedJPEG(3000, 3000, 4),
		"16-bit interlaced PNG of 40 MP": craftedPNG(t, 8000, 5000, true),
		"16-bit PNG of 26 MP":            craftedPNG(t, 6000, 4400, false),
	} {
		var ok bool
		if n := allocated(func() { _, ok = decodeImage(file) }); ok || n > 1<<20 {
			t.Errorf("%s: decoded %t, allocating %d bytes; want refused from the header", name, ok, n)
		}
	}
}

// decodeCost bounds what the decoder allocates, by format: measured on crafted images of every kind it budgets.
func TestDecodeCost_BoundsWhatTheDecoderAllocates(t *testing.T) {
	const w, h = 2000, 2000
	var baseline, rgba bytes.Buffer
	if err := jpeg.Encode(&baseline, image.NewYCbCr(image.Rect(0, 0, w, h), image.YCbCrSubsampleRatio444), nil); err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(&rgba, image.NewNRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	for name, file := range map[string][]byte{
		"baseline JPEG":          baseline.Bytes(),
		"progressive gray JPEG":  craftedJPEG(w, h, 1),
		"progressive YCbCr JPEG": craftedJPEG(w, h, 3),
		"progressive CMYK JPEG":  craftedJPEG(w, h, 4),
		"8-bit RGBA PNG":         rgba.Bytes(),
		"16-bit PNG":             craftedPNG(t, w, h, false),
		"16-bit interlaced PNG":  craftedPNG(t, w, h, true),
	} {
		cost, ok := decodeCost(file)
		n := allocated(func() {
			if _, _, err := image.Decode(bytes.NewReader(file)); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
		})
		if !ok || int64(n) > cost { //nolint:gosec // a test's allocation
			t.Errorf("%s: costed %d (%t), allocates %d", name, cost, ok, n)
		}
	}
}

// A phone's photo of 12 megapixels gets its thumbnail, progressive too, within the budget.
func TestDecodeImage_DecodesAPhonesProgressivePhoto(t *testing.T) {
	file := craftedJPEG(4000, 3000, 3)
	var ok bool
	if n := allocated(func() { _, ok = decodeImage(file) }); !ok || n > maxDecodeBytes {
		t.Errorf("decoded %t, allocating %d bytes; want decoded within %d", ok, n, maxDecodeBytes)
	}
}

// The worker decodes one image at a time, so that two scans never hold two budgets.
func TestScaledImage_DecodesOneImageAtATime(t *testing.T) {
	t.Parallel()
	var small bytes.Buffer
	if err := jpeg.Encode(&small, image.NewGray(image.Rect(0, 0, 10, 10)), nil); err != nil {
		t.Fatal(err)
	}
	s := &Service{}
	s.decoding.Lock()
	done := make(chan bool)
	go func() {
		_, ok := s.scaledImage(small.Bytes())
		done <- ok
	}()
	select {
	case <-done:
		t.Fatal("decoded while another decode runs")
	case <-time.After(50 * time.Millisecond):
	}
	s.decoding.Unlock()
	if !<-done {
		t.Error("the small JPEG: refused")
	}
}
