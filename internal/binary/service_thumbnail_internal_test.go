package binary

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
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
