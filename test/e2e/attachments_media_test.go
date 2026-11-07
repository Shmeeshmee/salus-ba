//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	mathbits "math/bits"
	"strconv"
	"testing"
)

// The files the attachment journeys upload, generated here: nothing in them comes from a real photo, document or
// video, and the EICAR test string is assembled, never stored as a file (docs/attachments/SPEC.md §5).

// photo is a generated 64x64 JPEG.
func photo(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for x := range 64 {
		for y := range 64 {
			img.Set(x, y, color.RGBA{R: 40, G: byte(120 + y), B: byte(x * 4), A: 255})
		}
	}
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// eicarPDF is a generated PDF whose one content stream is the EICAR anti-malware test string, uncompressed: the
// upload takes it as a document, and clamd finds the signature in it.
func eicarPDF() []byte {
	var s []byte
	for _, p := range []string{`X5O!P%@AP[4\PZX54(P^)7CC)7}`, `$EICAR-STANDARD-ANTIVIRUS`, `-TEST-FILE!$H+H*`} {
		s = append(s, p...)
	}
	return []byte("%PDF-1.4\n1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n" +
		"2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj\n" +
		"3 0 obj << /Type /Page /Parent 2 0 R /Contents 4 0 R >> endobj\n" +
		"4 0 obj << /Length " + strconv.Itoa(len(s)) + " >> stream\n" + string(s) + "\nendstream endobj\n" +
		"trailer << /Root 1 0 R >>\n%%EOF\n")
}

// The generated video: 128x96 pixels, 10 frames a second for 6 seconds, a bright bar moving across a dark
// picture. A real H.264 stream (Baseline, every frame an IDR picture of I_PCM macroblocks, which carry their
// pixels uncoded), so a browser decodes and seeks it; no audio track, which the server allows (SPEC §3 O18).
const (
	videoWidth   = 128
	videoHeight  = 96
	videoFPS     = 10
	videoFrames  = 60
	videoSeconds = videoFrames / videoFPS
)

// video is the generated MP4 of size bytes: ftyp, moov, a free box of a repeating counter that brings it to size,
// so a misplaced range shows, then the mdat with the frames. The frames lie past the filler, so a browser that plays
// the file has to ask for them with a Range request that starts past 0 (test/e2e/web/attachments.spec.ts).
func video(size int) []byte {
	sps, pps := videoSPS(), videoPPS()
	samples := make([][]byte, videoFrames)
	for i := range samples {
		nal := nalUnit(5, videoSlice(i))
		samples[i] = append(be32(uint32(len(nal))), nal...)
	}
	ftyp := mp4Box("ftyp", []byte("isom"), be32(512), []byte("isomiso2avc1mp41"))
	moov := videoMoov(sps, pps, samples, 0)
	mdat := mp4Box("mdat", samples...)
	pad := size - len(ftyp) - len(moov) - len(mdat) - 8
	if pad < 0 {
		panic("the generated video is larger than the size asked")
	}
	filler := make([]byte, pad)
	for i := range filler {
		filler[i] = byte(i * 7)
	}
	free := mp4Box("free", filler)
	offset := uint32(len(ftyp) + len(moov) + len(free) + 8)
	moov = videoMoov(sps, pps, samples, offset) // the same length: only the chunk offset changes
	return bytes.Join([][]byte{ftyp, moov, free, mdat}, nil)
}

func videoMoov(sps, pps []byte, samples [][]byte, offset uint32) []byte {
	const timescale, duration = 1000, videoSeconds * 1000
	matrix := bytes.Join([][]byte{be32(0x10000), be32(0), be32(0), be32(0), be32(0x10000), be32(0), be32(0), be32(0), be32(0x40000000)}, nil)
	mvhd := mp4Box("mvhd", be32(0), be32(0), be32(0), be32(timescale), be32(duration), be32(0x10000), be16(0x100), make([]byte, 10),
		matrix, make([]byte, 24), be32(2))
	tkhd := mp4Box("tkhd", be32(3), be32(0), be32(0), be32(1), be32(0), be32(duration), make([]byte, 8), be16(0), be16(0), be16(0), be16(0),
		matrix, be32(videoWidth<<16), be32(videoHeight<<16))
	mdhd := mp4Box("mdhd", be32(0), be32(0), be32(0), be32(timescale), be32(duration), be16(0x55c4), be16(0))
	hdlr := mp4Box("hdlr", be32(0), be32(0), []byte("vide"), make([]byte, 12), []byte("VideoHandler\x00"))
	avcC := mp4Box("avcC", []byte{1, 66, 0xc0, 30, 0xff, 0xe1}, be16(uint16(len(sps))), sps, []byte{1}, be16(uint16(len(pps))), pps)
	avc1 := mp4Box("avc1", make([]byte, 6), be16(1), make([]byte, 16), be16(videoWidth), be16(videoHeight), be32(0x480000), be32(0x480000),
		be32(0), be16(1), make([]byte, 32), be16(0x18), be16(0xffff), avcC)
	sizes := [][]byte{be32(0), be32(0), be32(uint32(len(samples)))}
	for _, s := range samples {
		sizes = append(sizes, be32(uint32(len(s))))
	}
	stbl := mp4Box("stbl",
		mp4Box("stsd", be32(0), be32(1), avc1),
		mp4Box("stts", be32(0), be32(1), be32(uint32(len(samples))), be32(timescale/videoFPS)),
		mp4Box("stsc", be32(0), be32(1), be32(1), be32(uint32(len(samples))), be32(1)),
		mp4Box("stsz", sizes...),
		mp4Box("stco", be32(0), be32(1), be32(offset)))
	dinf := mp4Box("dinf", mp4Box("dref", be32(0), be32(1), mp4Box("url ", be32(1))))
	minf := mp4Box("minf", mp4Box("vmhd", be32(1), make([]byte, 8)), dinf, stbl)
	return mp4Box("moov", mvhd, mp4Box("trak", tkhd, mp4Box("mdia", mdhd, hdlr, minf)))
}

// videoSPS is the sequence parameter set: Baseline, level 3.0, frame numbers of 4 bits, picture order from the
// frame number (type 2), one reference frame, the picture size in macroblocks.
func videoSPS() []byte {
	var w bitWriter
	w.u(66, 8)   // profile_idc: Baseline
	w.u(0xc0, 8) // constraint_set0 and constraint_set1
	w.u(30, 8)   // level_idc
	w.ue(0)      // seq_parameter_set_id
	w.ue(0)      // log2_max_frame_num_minus4
	w.ue(2)      // pic_order_cnt_type
	w.ue(1)      // max_num_ref_frames
	w.u(0, 1)    // gaps_in_frame_num_value_allowed_flag
	w.ue(videoWidth/16 - 1)
	w.ue(videoHeight/16 - 1)
	w.u(1, 1) // frame_mbs_only_flag
	w.u(1, 1) // direct_8x8_inference_flag
	w.u(0, 1) // frame_cropping_flag
	w.u(0, 1) // vui_parameters_present_flag
	w.trailing()
	return nalUnit(7, w.b)
}

// videoPPS is the picture parameter set: CAVLC, one slice group, no weighted prediction, QP 26.
func videoPPS() []byte {
	var w bitWriter
	w.ue(0)   // pic_parameter_set_id
	w.ue(0)   // seq_parameter_set_id
	w.u(0, 1) // entropy_coding_mode_flag: CAVLC
	w.u(0, 1) // bottom_field_pic_order_in_frame_present_flag
	w.ue(0)   // num_slice_groups_minus1
	w.ue(0)   // num_ref_idx_l0_default_active_minus1
	w.ue(0)   // num_ref_idx_l1_default_active_minus1
	w.u(0, 1) // weighted_pred_flag
	w.u(0, 2) // weighted_bipred_idc
	w.se(0)   // pic_init_qp_minus26
	w.se(0)   // pic_init_qs_minus26
	w.se(0)   // chroma_qp_index_offset
	w.u(0, 1) // deblocking_filter_control_present_flag
	w.u(0, 1) // constrained_intra_pred_flag
	w.u(0, 1) // redundant_pic_cnt_present_flag
	w.trailing()
	return nalUnit(8, w.b)
}

// videoSlice is frame n as one IDR slice of I_PCM macroblocks: the luma dark with a bright bar 16 pixels wide
// that moves 4 pixels a frame, the chroma grey.
func videoSlice(n int) []byte {
	var w bitWriter
	w.ue(0)             // first_mb_in_slice
	w.ue(7)             // slice_type: I, every slice of the picture
	w.ue(0)             // pic_parameter_set_id
	w.u(0, 4)           // frame_num: an IDR picture's is 0
	w.ue(uint64(n % 2)) // idr_pic_id: two IDR pictures in a row differ
	w.u(0, 1)           // no_output_of_prior_pics_flag
	w.u(0, 1)           // long_term_reference_flag
	w.se(0)             // slice_qp_delta
	for mb := range (videoWidth / 16) * (videoHeight / 16) {
		w.ue(25) // mb_type: I_PCM
		w.align()
		x0 := (mb % (videoWidth / 16)) * 16
		for range 16 {
			for x := range 16 {
				luma := byte(60)
				if (x0+x+videoWidth-4*n%videoWidth)%videoWidth < 16 {
					luma = 220
				}
				w.bytes(luma)
			}
		}
		w.bytes(bytes.Repeat([]byte{128}, 128)...) // Cb and Cr, 8x8 each
	}
	w.trailing()
	return w.b
}

// nalUnit is a NAL unit of the type with nal_ref_idc 3 around the RBSP, with emulation prevention bytes.
func nalUnit(typ byte, rbsp []byte) []byte {
	out := []byte{0x60 | typ}
	zeros := 0
	for _, c := range rbsp {
		if zeros >= 2 && c <= 3 {
			out = append(out, 3)
			zeros = 0
		}
		out = append(out, c)
		if c == 0 {
			zeros++
		} else {
			zeros = 0
		}
	}
	return out
}

// bitWriter writes the bits of an RBSP, most significant first.
type bitWriter struct {
	b []byte
	n int // bits written
}

func (w *bitWriter) bit(v uint64) {
	if w.n%8 == 0 {
		w.b = append(w.b, 0)
	}
	if v&1 == 1 {
		w.b[len(w.b)-1] |= 0x80 >> (w.n % 8)
	}
	w.n++
}

func (w *bitWriter) u(v uint64, n int) {
	for i := n - 1; i >= 0; i-- {
		w.bit(v >> i)
	}
}

// ue is unsigned Exp-Golomb: as many zeros as v+1 has bits after the first, then v+1.
func (w *bitWriter) ue(v uint64) {
	n := mathbits.Len64(v + 1)
	w.u(0, n-1)
	w.u(v+1, n)
}

func (w *bitWriter) se(v int64) {
	if v > 0 {
		w.ue(uint64(2*v - 1))
	} else {
		w.ue(uint64(-2 * v))
	}
}

// bytes writes whole bytes; the writer is at a byte boundary.
func (w *bitWriter) bytes(p ...byte) {
	w.b = append(w.b, p...)
	w.n += 8 * len(p)
}

func (w *bitWriter) align() {
	for w.n%8 != 0 {
		w.bit(0)
	}
}

// trailing is rbsp_trailing_bits: a one, then zeros to the byte.
func (w *bitWriter) trailing() {
	w.bit(1)
	w.align()
}

func mp4Box(typ string, parts ...[]byte) []byte {
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

func be32(v uint32) []byte { return binary.BigEndian.AppendUint32(nil, v) }

func be16(v uint16) []byte { return binary.BigEndian.AppendUint16(nil, v) }
