import 'dart:io';
import 'dart:math' as math;
import 'dart:typed_data';

// A file's type read from its bytes, and every metadata a phone writes removed from a photo or a video before upload
// (docs/attachments/SPEC.md §2 "Image and video metadata"): location, device, times, XMP, IPTC, comments; the
// orientation kept. Pure Dart over the bytes, tested on generated files (test/file_prep_test.dart). Nothing here
// logs.

/// What a file's first bytes say it is.
enum Format { jpeg, png, heif, pdf, mp4, mov, unknown }

const _heif = {'heic', 'heix', 'hevc', 'hevx', 'heim', 'heis', 'mif1', 'msf1', 'avif'};

Format detect(List<int> head) {
  bool at(int i, List<int> sig) {
    if (head.length < i + sig.length) return false;
    for (var k = 0; k < sig.length; k++) {
      if (head[i + k] != sig[k]) return false;
    }
    return true;
  }

  if (at(0, const [0xFF, 0xD8, 0xFF])) return Format.jpeg;
  if (at(0, const [0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A])) return Format.png;
  if (at(4, 'ftyp'.codeUnits) && head.length >= 12) {
    final brand = String.fromCharCodes(head.sublist(8, 12));
    if (_heif.contains(brand)) return Format.heif;
    return brand == 'qt  ' ? Format.mov : Format.mp4;
  }
  // As the server reads a PDF: `%PDF-` within the first 1,024 bytes.
  final pdf = '%PDF-'.codeUnits;
  for (var i = 0; i < math.min(head.length, 1024); i++) {
    if (at(i, pdf)) return Format.pdf;
  }
  return Format.unknown;
}

// Photos.

/// The Exif orientation (1 to 8) as the only tag of a big-endian TIFF, or null for none or upright.
List<int>? _orientationTiff(int? orientation) {
  if (orientation == null || orientation <= 1 || orientation > 8) return null;
  return [0x4D, 0x4D, 0, 42, 0, 0, 0, 8, 0, 1, 0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, orientation, 0, 0, 0, 0, 0, 0];
}

/// The Orientation tag of IFD0 in the TIFF at [tiff], or null.
int? _orientation(Uint8List tiff) {
  if (tiff.length < 8) return null;
  final endian = switch (String.fromCharCodes(tiff.sublist(0, 2))) {
    'II' => Endian.little,
    'MM' => Endian.big,
    _ => null,
  };
  if (endian == null) return null;
  final d = ByteData.sublistView(tiff);
  final ifd = d.getUint32(4, endian);
  if (ifd + 2 > tiff.length) return null;
  final count = d.getUint16(ifd, endian);
  for (var i = 0; i < count; i++) {
    final e = ifd + 2 + i * 12;
    if (e + 12 > tiff.length) return null;
    if (d.getUint16(e, endian) == 0x0112) return d.getUint16(e + 8, endian);
  }
  return null;
}

bool _startsWith(Uint8List b, int at, String s) {
  if (b.length < at + s.length) return false;
  for (var i = 0; i < s.length; i++) {
    if (b[at + i] != s.codeUnitAt(i)) return false;
  }
  return true;
}

/// The JPEG without its APP segments and comments (Exif, XMP, IPTC, JFIF, maker data, previews), except the
/// colour profile (ICC) and Adobe's colour transform, which say how to show the colours; with a new Exif holding only
/// the orientation when the photo had one; and without anything after its end (a preview or depth map a phone
/// appends, with its own Exif). The picture's own bytes are passed through. Throws [FormatException].
Uint8List stripJpeg(Uint8List b) {
  if (b.length < 4 || b[0] != 0xFF || b[1] != 0xD8) throw const FormatException('not a JPEG');
  final body = BytesBuilder(copy: false);
  int? orientation;
  var i = 2;
  while (true) {
    if (i + 1 >= b.length || b[i] != 0xFF) throw const FormatException('broken JPEG');
    final m = b[i + 1];
    if (m == 0xFF) {
      i++;
      continue;
    }
    if (m == 0xD9) break;
    if (m == 0x01 || (m >= 0xD0 && m <= 0xD7)) {
      body.add(Uint8List.sublistView(b, i, i + 2));
      i += 2;
      continue;
    }
    if (i + 4 > b.length) throw const FormatException('broken JPEG');
    final end = i + 2 + (b[i + 2] << 8 | b[i + 3]);
    if (end > b.length) throw const FormatException('broken JPEG');
    if (m == 0xE1 && _startsWith(b, i + 4, 'Exif\u0000\u0000')) {
      orientation ??= _orientation(Uint8List.sublistView(b, i + 10, end));
    }
    if (_keepSegment(b, i, m)) body.add(Uint8List.sublistView(b, i, end));
    i = end;
    if (m == 0xDA) i = _afterScan(b, i, body);
  }
  final exif = _orientationTiff(orientation);
  final out = BytesBuilder(copy: false)..add(const [0xFF, 0xD8]);
  if (exif != null) {
    final payload = [...'Exif'.codeUnits, 0, 0, ...exif];
    out.add([0xFF, 0xE1, (payload.length + 2) >> 8, (payload.length + 2) & 0xFF, ...payload]);
  }
  return (out
        ..add(body.takeBytes())
        ..add(const [0xFF, 0xD9]))
      .takeBytes();
}

bool _keepSegment(Uint8List b, int i, int m) {
  if (m == 0xFE) return false;
  if (m < 0xE0 || m > 0xEF) return true;
  return (m == 0xE2 && _startsWith(b, i + 4, 'ICC_PROFILE\u0000')) || (m == 0xEE && _startsWith(b, i + 4, 'Adobe'));
}

/// Copies the entropy-coded bytes after a scan header at [i] up to the next marker (a stuffed 0xFF00 and the
/// restart markers are data), and answers where that marker is.
int _afterScan(Uint8List b, int i, BytesBuilder body) {
  var j = i;
  while (j + 1 < b.length) {
    final next = b[j + 1];
    if (b[j] == 0xFF && next != 0x00 && !(next >= 0xD0 && next <= 0xD7)) break;
    j++;
  }
  if (j + 1 >= b.length) throw const FormatException('broken JPEG');
  body.add(Uint8List.sublistView(b, i, j));
  return j;
}

/// The chunks a PNG needs to be shown as it was taken; every other one (text, XMP, time, Exif) goes.
const _pngKept = {
  'IHDR',
  'PLTE',
  'IDAT',
  'IEND',
  'tRNS',
  'cHRM',
  'gAMA',
  'iCCP',
  'sBIT',
  'sRGB',
  'bKGD',
  'pHYs',
  'acTL',
  'fcTL', //
  'fdAT',
};

/// The PNG with only the chunks of [_pngKept], and an `eXIf` holding only the orientation when it had one.
/// Throws [FormatException].
Uint8List stripPng(Uint8List b) {
  if (detect(b) != Format.png) throw const FormatException('not a PNG');
  final d = ByteData.sublistView(b);
  final chunks = <(String, Uint8List)>[];
  int? orientation;
  for (var i = 8; i < b.length;) {
    if (i + 12 > b.length) throw const FormatException('broken PNG');
    final len = d.getUint32(i);
    if (i + 12 + len > b.length) throw const FormatException('broken PNG');
    final type = String.fromCharCodes(b.sublist(i + 4, i + 8));
    if (type == 'eXIf') orientation = _orientation(Uint8List.sublistView(b, i + 8, i + 8 + len));
    if (_pngKept.contains(type)) chunks.add((type, Uint8List.sublistView(b, i, i + 12 + len)));
    i += 12 + len;
  }
  final out = BytesBuilder(copy: false)..add(Uint8List.sublistView(b, 0, 8));
  final exif = _orientationTiff(orientation);
  for (final (type, chunk) in chunks) {
    out.add(chunk);
    if (type == 'IHDR' && exif != null) {
      final body = [...'eXIf'.codeUnits, ...exif];
      out
        ..add(_u32(exif.length))
        ..add(body)
        ..add(_u32(crc32(body)));
    }
  }
  return out.takeBytes();
}

List<int> _u32(int v) => [v >> 24 & 0xFF, v >> 16 & 0xFF, v >> 8 & 0xFF, v & 0xFF];

final _crcTable = List<int>.generate(256, (n) {
  var c = n;
  for (var k = 0; k < 8; k++) {
    c = c & 1 != 0 ? 0xEDB88320 ^ (c >> 1) : c >> 1;
  }
  return c;
});

/// The CRC-32 a PNG chunk carries over its type and data.
int crc32(List<int> bytes) {
  var c = 0xFFFFFFFF;
  for (final byte in bytes) {
    c = _crcTable[(c ^ byte) & 0xFF] ^ (c >> 8);
  }
  return c ^ 0xFFFFFFFF;
}

// Videos.

/// Bytes to write over a file at an offset; the same length as what they replace.
typedef Patch = ({int offset, Uint8List bytes});

/// What a movie's boxes say: its brand, the sample entries of its tracks (`avc1`, `mp4a`, `hvc1`…), and the patches
/// that remove its metadata without moving a byte of its media.
class Movie {
  const Movie({required this.brand, required this.entries, required this.patches});

  final String brand;
  final Set<String> entries;
  final List<Patch> patches;

  static const _iso = {'isom', 'iso2', 'iso4', 'iso5', 'iso6', 'mp41', 'mp42', 'avc1'};

  /// An MP4 with H.264 video and AAC sound or none, as the server accepts it (SPEC §3 O1).
  bool get playsInChrome =>
      _iso.contains(brand) && entries.contains('avc1') && entries.every(const {'avc1', 'mp4a'}.contains);
}

/// The movie's `moov` is read into memory; past this it is refused, as on the server (package 1.2).
const _maxMoov = 16 * 1024 * 1024;

/// Boxes that hold metadata, at the top, in `moov` or in a `trak`: user data (location `©xyz`, device `©mak` and
/// `©mod`), Apple's metadata keys, and XMP in a `uuid` box. Each becomes a `free` box of zeros.
const _metadataBoxes = {'udta', 'meta', 'uuid'};

/// Reads the box structure of the movie at [f]. Throws [FormatException] for a box that lies about its size.
Future<Movie> readMovie(File f) async {
  final raf = await f.open();
  try {
    final length = await raf.length();
    var brand = '';
    final entries = <String>{};
    final patches = <Patch>[];
    for (var at = 0; at < length;) {
      await raf.setPosition(at);
      final head = Uint8List.fromList(await raf.read(16));
      final (size, header) = _boxSize(head, length - at);
      final type = String.fromCharCodes(head.sublist(4, 8));
      if (type == 'ftyp' && head.length >= 12) brand = String.fromCharCodes(head.sublist(8, 12));
      if (_metadataBoxes.contains(type)) patches.addAll(_freed(at, header, size));
      if (type == 'moov') {
        if (size > _maxMoov) throw const FormatException('moov too large');
        await raf.setPosition(at);
        final moov = Uint8List.fromList(await raf.read(size));
        _walk(moov, header, size, at, 'moov', entries, patches);
      }
      at += size;
    }
    return Movie(brand: brand, entries: entries, patches: patches);
  } finally {
    await raf.close();
  }
}

/// A box's size and header length from its first bytes; [left] is what the file or parent box still holds.
(int, int) _boxSize(Uint8List head, int left) {
  if (head.length < 8) throw const FormatException('broken box');
  final d = ByteData.sublistView(head);
  var size = d.getUint32(0);
  var header = 8;
  if (size == 1) {
    if (head.length < 16) throw const FormatException('broken box');
    size = d.getUint64(8);
    header = 16;
  } else if (size == 0) {
    size = left;
  }
  if (size < header || size > left) throw const FormatException('box size');
  return (size, header);
}

/// The box at [offset] made a `free` box of zeros.
List<Patch> _freed(int offset, int header, int size) => [
  (offset: offset + 4, bytes: Uint8List.fromList('free'.codeUnits)),
  (offset: offset + header, bytes: Uint8List(size - header)),
];

const _containers = {'trak', 'mdia', 'minf', 'stbl'};

/// Walks the children of the box [parent] in [b] from [start] to [end]; [base] is [b]'s offset in the file.
void _walk(Uint8List b, int start, int end, int base, String parent, Set<String> entries, List<Patch> patches) {
  for (var at = start; at < end;) {
    final (size, header) = _boxSize(Uint8List.sublistView(b, at, math.min(at + 16, end)), end - at);
    final type = String.fromCharCodes(b.sublist(at + 4, at + 8));
    if (_metadataBoxes.contains(type) && (parent == 'moov' || parent == 'trak')) {
      patches.addAll(_freed(base + at, header, size));
    } else if (const {'mvhd', 'tkhd', 'mdhd'}.contains(type) && size >= header + 4) {
      // Creation and modification times: 32 bits each in version 0, 64 in version 1. The rest (the track's
      // matrix, its orientation) stays.
      final times = b[at + header] == 1 ? 16 : 8;
      if (size >= header + 4 + times) patches.add((offset: base + at + header + 4, bytes: Uint8List(times)));
    } else if (type == 'stsd') {
      _sampleEntries(b, at + header, at + size, entries);
    } else if (_containers.contains(type)) {
      _walk(b, at + header, at + size, base, type, entries, patches);
    }
    at += size;
  }
}

/// What [entries] holds for an `stsd` whose count is not the number of entries in it: no sample entry has this
/// name, so the movie does not play in Chrome and is converted.
const _miscounted = 'stsd count';

/// Adds the sample entries of the `stsd` payload from [start] to [end]. The server takes the payload only when it is
/// exactly the entries it counts (pkg/filetype/mp4.go), so a count that differs makes the movie one to convert.
void _sampleEntries(Uint8List b, int start, int end, Set<String> entries) {
  if (start + 8 > end) throw const FormatException('broken stsd');
  final count = ByteData.sublistView(b).getUint32(start + 4);
  var found = 0;
  for (var at = start + 8; at < end; found++) {
    final (size, _) = _boxSize(Uint8List.sublistView(b, at, math.min(at + 16, end)), end - at);
    entries.add(String.fromCharCodes(b.sublist(at + 4, at + 8)));
    at += size;
  }
  if (found != count) entries.add(_miscounted);
}

/// Copies [src] to [dst] in chunks, with [patches] written over it: the memory used is one chunk.
Future<void> writePatched(File src, File dst, List<Patch> patches) async {
  const chunk = 1 << 20;
  final input = await src.open();
  final output = await dst.open(mode: FileMode.write);
  try {
    final length = await input.length();
    for (var at = 0; at < length; at += chunk) {
      final bytes = Uint8List.fromList(await input.read(chunk));
      final end = at + bytes.length;
      for (final p in patches) {
        final from = math.max(at, p.offset);
        final to = math.min(end, p.offset + p.bytes.length);
        if (from < to) bytes.setRange(from - at, to - at, p.bytes, from - p.offset);
      }
      await output.writeFrom(bytes);
    }
  } finally {
    await input.close();
    await output.close();
  }
}
