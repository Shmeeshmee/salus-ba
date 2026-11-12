package binary

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png" // decodes the PNGs a chat image may be
	"io"

	"golang.org/x/image/draw"

	"github.com/Shmeeshmee/salus/pkg/envelope"
)

// ThumbnailEdge is the longest side of a chat image's thumbnail, in pixels: a message's preview on a phone's screen
// and in the web app's list (SAL-30). A smaller image keeps its size.
const ThumbnailEdge = 320

// thumbnailType is what a thumbnail is: a JPEG, whatever the image.
const thumbnailType = "image/jpeg"

// maxThumbnailPixels bounds the images a thumbnail is made of, whatever their format; a larger one gets none.
const maxThumbnailPixels = 40_000_000

// maxDecodeBytes bounds the memory one decode may take in the worker, as decodeCost reckons it from the header: the
// standard library's decoders take from 1.5 bytes a pixel (a baseline 4:2:0 JPEG) to 24 (a progressive CMYK JPEG),
// so a crafted file of a few hundred kilobytes would take 600 MB at maxThumbnailPixels (BenchmarkDecodeCrafted,
// docs/plans/chat.PROGRESS.md). At this bound a phone's progressive photo of 12 megapixels (15 bytes a pixel) still
// gets its thumbnail; a file past it gets none, and is clean all the same.
const maxDecodeBytes = 192 << 20

// decodeOverhead is what a decode allocates beside its pixels and coefficients: tables, zlib's window, row buffers.
const decodeOverhead = 1 << 20

// thumbnailQuality is the thumbnail's JPEG quality: a preview, not the file.
const thumbnailQuality = 80

// thumbnail makes the thumbnail of the clean chat file b when it is an image (docs/plans/chat.md, SAL-30): its
// plaintext decoded in the worker, scaled to ThumbnailEdge, encoded as a JPEG and sealed into the store under a new id
// with its own data key, wrapped like the file's. A zero Thumbnail for a PDF, and for an image whose pixels do not
// decode or exceed maxThumbnailPixels: the file is clean all the same, its message shows it without a preview. The
// image's orientation tag is not applied (the apps send images upright, docs/attachments/SPEC.md §2).
func (s *Service) thumbnail(ctx context.Context, b Binary) (Thumbnail, error) {
	if kindOf(b.ContentType) != KindPhoto {
		return Thumbnail{}, nil
	}
	plain, err := s.plaintext(ctx, b)
	if err != nil {
		return Thumbnail{}, err
	}
	small, ok := s.scaledImage(plain)
	if !ok {
		return Thumbnail{}, nil
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, small, &jpeg.Options{Quality: thumbnailQuality}); err != nil {
		return Thumbnail{}, err
	}
	t := Thumbnail{ID: newID(), Size: int64(out.Len())}
	dek, err := envelope.NewDataKey()
	if err != nil {
		return Thumbnail{}, err
	}
	if t.Key, err = s.KEK.Wrap(dek, t.ID); err != nil {
		return Thumbnail{}, err
	}
	if _, err := s.put(ctx, Binary{ID: t.ID, Size: t.Size}, dek, &out, false); err != nil {
		return Thumbnail{}, err
	}
	return t, nil
}

// plaintext reads the whole plaintext of b from the store (a chat image: at most MaxPhotoSize bytes).
func (s *Service) plaintext(ctx context.Context, b Binary) ([]byte, error) {
	dek, err := s.KEK.Unwrap(b.WrappedKey, b.ID)
	if err != nil {
		return nil, err
	}
	body, err := s.Store.GetRange(ctx, b.ID, 0, envelope.CiphertextSize(b.Size)-1)
	if err != nil {
		return nil, err
	}
	defer func() { _ = body.Close() }()
	plain, err := envelope.Open(body, dek, b.ID)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(io.LimitReader(plain, b.Size))
}

// scaledImage is the image of the file b scaled to ThumbnailEdge, or false (decodeImage). The worker decodes one
// image at a time, so its scans never hold more than one maxDecodeBytes between them.
func (s *Service) scaledImage(b []byte) (image.Image, bool) {
	s.decoding.Lock()
	defer s.decoding.Unlock()
	img, ok := decodeImage(b)
	if !ok {
		return nil, false
	}
	return scaled(img, ThumbnailEdge), true
}

// decodeImage decodes a JPEG or PNG of at most maxThumbnailPixels pixels whose decode costs at most maxDecodeBytes;
// false for anything else, refused from its header before a pixel is decoded.
func decodeImage(b []byte) (image.Image, bool) {
	if cost, ok := decodeCost(b); !ok || cost > maxDecodeBytes {
		return nil, false
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	return img, err == nil
}

// decodeCost is what the standard library's decoder allocates for the JPEG or PNG b, at most, from its header
// (measured, TestDecodeCost_BoundsWhatTheDecoderAllocates): its pixels times the bytes a pixel of its format; false
// for anything else and for an image past maxThumbnailPixels.
func decodeCost(b []byte) (int64, bool) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return 0, false
	}
	pixels := int64(cfg.Width) * int64(cfg.Height)
	if pixels > maxThumbnailPixels {
		return 0, false
	}
	var perPixel int64
	switch format {
	case "jpeg":
		perPixel = jpegBytes(cfg.ColorModel, progressive(b))
	case "png":
		perPixel = pngBytes(cfg.ColorModel, len(b) > 28 && b[28] == 1) // IHDR's interlace method, Adam7
	default:
		return 0, false
	}
	return pixels*perPixel + decodeOverhead, true
}

// jpegBytes is what a JPEG's decode takes a pixel: a byte a component, without counting its subsampling; four more
// for the RGBA or CMYK image an RGB or CMYK one is converted to; and, progressive, an int32 a coefficient of every
// component, held until the last scan.
func jpegBytes(model color.Model, progressive bool) int64 {
	components, converted := int64(3), int64(0)
	switch model {
	case color.GrayModel:
		components = 1
	case color.CMYKModel:
		components, converted = 4, 4
	case color.RGBAModel:
		converted = 4
	}
	n := components + converted
	if progressive {
		n += 4 * components
	}
	return n
}

// pngBytes is what a PNG's decode takes a pixel: its image's, twice when interlaced (each pass, then the whole).
func pngBytes(model color.Model, interlaced bool) int64 {
	var n int64
	switch model {
	case color.GrayModel:
		n = 1
	case color.Gray16Model:
		n = 2
	case color.RGBAModel, color.NRGBAModel:
		n = 4
	default:
		if _, ok := model.(color.Palette); ok {
			n = 1
		} else {
			n = 8 // RGBA64 and NRGBA64: 16 bits a channel
		}
	}
	if interlaced {
		n *= 2
	}
	return n
}

// progressive is whether the JPEG b is progressive (its frame a SOF2), from its markers; true when they do not say,
// so that a file the walk does not follow is costed at its dearest.
func progressive(b []byte) bool {
	for i := 2; i+4 <= len(b); {
		if b[i] != 0xFF {
			return true
		}
		switch m := b[i+1]; {
		case m == 0xFF: // fill
			i++
			continue
		case m == 0xC2:
			return true
		case m >= 0xC0 && m <= 0xCF && m != 0xC4 && m != 0xC8 && m != 0xCC: // another frame
			return false
		}
		i += 2 + (int(b[i+2])<<8 | int(b[i+3]))
	}
	return true
}

// scaled is img with its longest side at most edge pixels, its proportions kept (bilinear, over every source pixel).
func scaled(img image.Image, edge int) image.Image {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	if w <= edge && h <= edge {
		return img
	}
	if w >= h {
		w, h = edge, max(1, h*edge/w)
	} else {
		w, h = max(1, w*edge/h), edge
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.BiLinear.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Src, nil)
	return dst
}
