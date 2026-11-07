package filetype

import (
	"encoding/binary"
	"math"
)

// maxMoov is the largest moov box the Checker buffers, header included. A phone's ten-minute video has a
// moov of a few megabytes.
const maxMoov = 16 << 20

// Checker checks an MP4 as it streams through it, from its first byte: the top-level boxes must add up to
// the file, the first is ftyp, there is exactly one moov (before or after the media data), and every sample
// entry of every track is avc1 (H.264) or mp4a (AAC), with at least one video track. Only moov is buffered;
// it is checked as soon as it is complete. Write fails with the first error and keeps failing; Close
// reports a file that ends inside a box or has no moov. A Checker is used for one file.
type Checker struct {
	err     error
	moov    []byte
	left    int64 // bytes of the current box still to come
	header  [16]byte
	boxes   int
	headerN int
	inBox   bool
	toEnd   bool // the current box runs to the end of the file
	inMoov  bool
	sawMoov bool
}

// NewChecker returns a Checker for one file.
func NewChecker() *Checker { return &Checker{} }

// Write takes the next bytes of the file.
func (c *Checker) Write(p []byte) (int, error) {
	if c.err != nil {
		return 0, c.err
	}
	n := len(p)
	for len(p) > 0 && c.err == nil {
		if c.inBox {
			p = c.payload(p)
		} else {
			p = c.readHeader(p)
		}
	}
	if c.err != nil {
		return 0, c.err
	}
	return n, nil
}

// Close checks that the file ended at the end of a box and held a moov.
func (c *Checker) Close() error {
	if c.err == nil && (c.headerN > 0 || (c.inBox && !c.toEnd) || !c.sawMoov) {
		c.err = ErrMalformed
	}
	return c.err
}

// readHeader collects the next box header (8 bytes, or 16 with a 64-bit size) and starts the box.
func (c *Checker) readHeader(p []byte) []byte {
	want := 8
	if c.headerN >= 4 && binary.BigEndian.Uint32(c.header[:]) == 1 {
		want = 16
	}
	k := copy(c.header[c.headerN:want], p)
	c.headerN += k
	p = p[k:]
	if c.headerN == 8 && want == 8 && binary.BigEndian.Uint32(c.header[:]) == 1 {
		return p // a 64-bit size follows
	}
	if c.headerN < want {
		return p
	}
	c.start(want)
	return p
}

// start begins the box whose header of headerLen bytes is complete.
func (c *Checker) start(headerLen int) {
	typ := string(c.header[4:8])
	size, toEnd, ok := boxSize(c.header[:headerLen])
	c.headerN = 0
	c.boxes++
	switch {
	case !ok,
		c.boxes == 1 && typ != "ftyp",
		toEnd && (typ == "ftyp" || typ == "moov"),
		typ == "moov" && (c.sawMoov || size > maxMoov):
		c.err = ErrMalformed
		return
	}
	c.inBox, c.toEnd, c.left = true, toEnd, size-int64(headerLen)
	c.inMoov = typ == "moov"
	c.sawMoov = c.sawMoov || c.inMoov
	if c.left == 0 && !toEnd {
		c.finish()
	}
}

// payload consumes the current box's bytes, buffering them only for moov.
func (c *Checker) payload(p []byte) []byte {
	if c.toEnd {
		return nil
	}
	k := int(min(int64(len(p)), c.left))
	if c.inMoov {
		c.moov = append(c.moov, p[:k]...)
	}
	c.left -= int64(k)
	if c.left == 0 {
		c.finish()
	}
	return p[k:]
}

// finish ends the current box and checks it when it is moov.
func (c *Checker) finish() {
	c.inBox = false
	if c.inMoov {
		c.inMoov = false
		c.err = checkMoov(c.moov)
		c.moov = nil
	}
}

// boxSize reads a box header: its total size, whether it runs to the end of its container (size 0), and
// whether the size is at least the header's length.
func boxSize(header []byte) (size int64, toEnd, ok bool) {
	switch s := binary.BigEndian.Uint32(header); s {
	case 0:
		return 0, true, true
	case 1:
		if len(header) < 16 {
			return 0, false, false
		}
		large := binary.BigEndian.Uint64(header[8:])
		return int64(min(large, math.MaxInt64)), false, large >= 16
	default:
		return int64(s), false, s >= 8
	}
}

// children calls fn with the type and payload of each box in b, which must be exactly a sequence of boxes.
func children(b []byte, fn func(typ string, payload []byte) error) error {
	for len(b) > 0 {
		if len(b) < 8 {
			return ErrMalformed
		}
		headerLen := 8
		if binary.BigEndian.Uint32(b) == 1 {
			headerLen = 16
		}
		if len(b) < headerLen {
			return ErrMalformed
		}
		size, toEnd, ok := boxSize(b[:headerLen])
		if toEnd {
			size = int64(len(b))
		}
		if !ok || size > int64(len(b)) {
			return ErrMalformed
		}
		if err := fn(string(b[4:8]), b[headerLen:size]); err != nil {
			return err
		}
		b = b[size:]
	}
	return nil
}

// child returns the payload of b's first child box of type typ.
func child(b []byte, typ string) ([]byte, error) {
	var found []byte
	err := children(b, func(t string, payload []byte) error {
		if t == typ && found == nil {
			found = payload
		}
		return nil
	})
	if err == nil && found == nil {
		err = ErrMalformed
	}
	return found, err
}

// checkMoov checks the sample entries of every track in a moov payload.
func checkMoov(moov []byte) error {
	video := false
	err := children(moov, func(typ string, trak []byte) error {
		if typ != "trak" {
			return nil
		}
		hasVideo, err := checkTrack(trak)
		video = video || hasVideo
		return err
	})
	if err == nil && !video {
		err = ErrUnsupported
	}
	return err
}

// checkTrack follows trak → mdia → minf → stbl → stsd and accepts only avc1 and mp4a sample entries. It
// reports whether the track holds video.
func checkTrack(trak []byte) (bool, error) {
	b := trak
	for _, typ := range []string{"mdia", "minf", "stbl", "stsd"} {
		var err error
		if b, err = child(b, typ); err != nil {
			return false, err
		}
	}
	if len(b) < 8 {
		return false, ErrMalformed
	}
	want := binary.BigEndian.Uint32(b[4:]) // after version and flags
	var count uint32
	video := false
	err := children(b[8:], func(typ string, _ []byte) error {
		count++
		switch typ {
		case "avc1":
			video = true
		case "mp4a":
		default:
			return ErrUnsupported
		}
		return nil
	})
	if err == nil && count != want {
		err = ErrMalformed
	}
	return video, err
}
