package filetype

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

// check streams file through a Checker in writes of the given size and returns Close's error, or the
// first Write error.
func check(file []byte, chunk int) error {
	c := NewChecker()
	for len(file) > 0 {
		n := min(chunk, len(file))
		if _, err := c.Write(file[:n]); err != nil {
			return err
		}
		file = file[n:]
	}
	return c.Close()
}

var chunks = []int{1, 7, 4096, 1 << 30}

func TestChecker_Accepts(t *testing.T) {
	tests := []struct {
		name string
		file []byte
	}{
		{"moov first", mp4(true, trak("avc1"), trak("mp4a"))},
		{"moov last", mp4(false, trak("avc1"), trak("mp4a"))},
		{"without audio", mp4(true, trak("avc1"))},
		{"two video sample entries", mp4(false, trak("avc1", "avc1"), trak("mp4a"))},
		{"64-bit box size", cat(ftyp("isom"), moov(trak("avc1")), largeBox("mdat", make([]byte, 300)))},
		{"last box to the end of the file", cat(ftyp("isom"), moov(trak("avc1")), []byte{0, 0, 0, 0, 'm', 'd', 'a', 't'}, make([]byte, 300))},
		{"64-bit child size inside moov", cat(ftyp("isom"), box("moov", largeBox("free", nil), trak("avc1")))},
	}
	for _, tt := range tests {
		for _, chunk := range chunks {
			if err := check(tt.file, chunk); err != nil {
				t.Errorf("%s, writes of %d: %v", tt.name, chunk, err)
			}
		}
	}
}

func TestChecker_RefusesCodecs(t *testing.T) {
	tests := []struct {
		name string
		file []byte
	}{
		{"hevc", mp4(true, trak("hvc1"), trak("mp4a"))},
		{"hevc as hev1", mp4(false, trak("hev1"))},
		{"opus audio", mp4(true, trak("avc1"), trak("Opus"))},
		{"a second, unsupported sample entry", mp4(true, trak("avc1", "hvc1"))},
		{"encrypted video", mp4(true, trak("encv"))},
		{"audio only", mp4(true, trak("mp4a"))},
		{"no tracks", mp4(true)},
	}
	for _, tt := range tests {
		for _, chunk := range chunks {
			if err := check(tt.file, chunk); !errors.Is(err, ErrUnsupported) {
				t.Errorf("%s, writes of %d: err = %v, want ErrUnsupported", tt.name, chunk, err)
			}
		}
	}
}

func TestChecker_RefusesMalformedFiles(t *testing.T) {
	good := mp4(true, trak("avc1"))
	lying := bytes.Clone(good)
	binary.BigEndian.PutUint32(lying[len(lying)-5008:], 1<<20) // the mdat claims a megabyte, 5,000 bytes follow

	childTooBig := cat(ftyp("isom"), box("moov", []byte{0, 0, 1, 0, 't', 'r', 'a', 'k'}))
	stsdCountLies := cat(ftyp("isom"), moov(bytes.Replace(trak("avc1"), []byte{0, 0, 0, 1, 0, 0, 0, 16, 'a'}, []byte{0, 0, 0, 2, 0, 0, 0, 16, 'a'}, 1)))

	tests := []struct {
		name string
		file []byte
	}{
		{"empty", nil},
		{"no ftyp first", cat(moov(trak("avc1")), ftyp("isom"))},
		{"no moov", cat(ftyp("isom"), mdat(100))},
		{"two moov", cat(ftyp("isom"), moov(trak("avc1")), moov(trak("avc1")))},
		{"a box size that lies", lying},
		{"cut inside a box header", good[:len(good)-5004]},
		{"box size below its header", cat(ftyp("isom"), []byte{0, 0, 0, 7, 'f', 'r', 'e', 'e'}, moov(trak("avc1")))},
		{"64-bit size below its header", cat(ftyp("isom"), []byte{0, 0, 0, 1, 'f', 'r', 'e', 'e', 0, 0, 0, 0, 0, 0, 0, 15}, moov(trak("avc1")))},
		{"moov to the end of the file", cat(ftyp("isom"), []byte{0, 0, 0, 0, 'm', 'o', 'o', 'v'}, trak("avc1"))},
		{"a child bigger than moov", childTooBig},
		{"a track without a sample description", cat(ftyp("isom"), moov(box("trak", box("tkhd", make([]byte, 84)))))},
		{"a sample description short of its count", stsdCountLies},
		{"a sample description without its header", cat(ftyp("isom"), moov(box("trak", box("mdia", box("minf", box("stbl", box("stsd", []byte{0, 0})))))))},
		{"a child cut inside its header", cat(ftyp("isom"), box("moov", []byte{0, 0, 0}))},
	}
	for _, tt := range tests {
		for _, chunk := range chunks {
			if err := check(tt.file, chunk); !errors.Is(err, ErrMalformed) {
				t.Errorf("%s, writes of %d: err = %v, want ErrMalformed", tt.name, chunk, err)
			}
		}
	}
}

func TestChecker_MoovUpToSixteenMegabytes(t *testing.T) {
	padding := func(moovSize int) []byte {
		inner := moov(trak("avc1"))[8:] // moov's children
		return box("free", make([]byte, moovSize-8-len(inner)-8))
	}
	atLimit := cat(ftyp("isom"), box("moov", moov(trak("avc1"))[8:], padding(16<<20)))
	if err := check(atLimit, 1<<16); err != nil {
		t.Fatalf("a moov of exactly 16 MiB: %v", err)
	}
	past := cat(ftyp("isom"), box("moov", moov(trak("avc1"))[8:], padding(16<<20+1)))
	if err := check(past, 1<<16); !errors.Is(err, ErrMalformed) {
		t.Fatalf("a moov past 16 MiB: err = %v, want ErrMalformed", err)
	}
}

func TestChecker_RefusesAnOversizedMoovFromItsHeader(t *testing.T) {
	// A header that declares a huge moov is refused before any of it is buffered.
	c := NewChecker()
	header := cat(ftyp("isom"), []byte{0, 0, 0, 1, 'm', 'o', 'o', 'v', 0, 0, 0, 1, 0, 0, 0, 0})
	if _, err := c.Write(header); !errors.Is(err, ErrMalformed) {
		t.Fatalf("err = %v, want ErrMalformed", err)
	}
	if cap(c.moov) != 0 {
		t.Fatalf("buffered %d bytes", cap(c.moov))
	}
}

func TestChecker_StaysFailed(t *testing.T) {
	c := NewChecker()
	if _, err := c.Write(mp4(true, trak("hvc1"))); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("err = %v", err)
	}
	if n, err := c.Write(mdat(10)); n != 0 || !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Write after a failure = %d, %v", n, err)
	}
	if err := c.Close(); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Close after a failure = %v", err)
	}
}

func TestChecker_BuffersOnlyMoov(t *testing.T) {
	c := NewChecker()
	if _, err := c.Write(cat(ftyp("isom"), mdat(1<<20))); err != nil {
		t.Fatal(err)
	}
	if cap(c.moov) != 0 {
		t.Fatalf("buffered %d bytes of mdat", cap(c.moov))
	}
	if _, err := c.Write(moov(trak("avc1"))); err != nil {
		t.Fatal(err)
	}
	if c.moov != nil {
		t.Fatalf("kept %d bytes of moov after checking it", cap(c.moov))
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

func FuzzChecker(f *testing.F) {
	f.Add(mp4(true, trak("avc1"), trak("mp4a")), uint16(7))
	f.Add(mp4(false, trak("avc1")), uint16(1))
	f.Add(mp4(true, trak("hvc1")), uint16(100))
	f.Add(cat(ftyp("isom"), largeBox("mdat", make([]byte, 20)), moov(trak("avc1"))), uint16(3))
	f.Add(cat(ftyp("isom"), []byte{0, 0, 0, 0, 'm', 'd', 'a', 't'}), uint16(5))
	f.Fuzz(func(t *testing.T, file []byte, chunk uint16) {
		c := NewChecker()
		rest := file
		for len(rest) > 0 {
			n := min(int(chunk)+1, len(rest))
			if _, err := c.Write(rest[:n]); err != nil {
				break
			}
			rest = rest[n:]
			// Memory is bounded by what arrived: nothing is allocated for a size a header only declares.
			if cap(c.moov) > 4*len(file)+64 {
				t.Fatalf("buffered %d bytes of a %d-byte file", cap(c.moov), len(file))
			}
		}
		err := c.Close()
		if err != nil && !errors.Is(err, ErrMalformed) && !errors.Is(err, ErrUnsupported) {
			t.Fatalf("unexpected error %v", err)
		}
	})
}
