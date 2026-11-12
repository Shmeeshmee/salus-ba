package binary

import (
	"bytes"
	"context"
	"image"
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

// maxThumbnailPixels bounds the images a thumbnail is made of: decoded, an image takes about four bytes a pixel in
// the worker (160 MB at this bound); a larger one gets none.
const maxThumbnailPixels = 40_000_000

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
	img, ok := decodeImage(plain)
	if !ok {
		return Thumbnail{}, nil
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, scaled(img, ThumbnailEdge), &jpeg.Options{Quality: thumbnailQuality}); err != nil {
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

// decodeImage decodes a JPEG or PNG of at most maxThumbnailPixels pixels; false for anything else.
func decodeImage(b []byte) (image.Image, bool) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > maxThumbnailPixels {
		return nil, false
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	return img, err == nil
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
