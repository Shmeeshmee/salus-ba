import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:salus/file_prep.dart';
import 'package:salus/metadata.dart';

// Every fixture is generated here: synthetic bytes that carry the metadata a phone writes (a GPS position, the
// device's make and model, a creation time, an orientation), never a real photo or video.

const _make = 'SyntheticPhoneMaker';
const _model = 'SyntheticPhone 1';
const _xmp = 'http://ns.adobe.com/xap/1.0/\u0000<x:xmpmeta>synthetic location</x:xmpmeta>';
const _iptc = 'Photoshop 3.0\u00008BIM synthetic caption';
const _comment = 'synthetic comment';
const _location = '+43.8563+018.4131/';

Uint8List _bytes(List<int> b) => Uint8List.fromList(b);
List<int> _u16(int v) => [v >> 8 & 0xFF, v & 0xFF];
List<int> _u32(int v) => [v >> 24 & 0xFF, v >> 16 & 0xFF, v >> 8 & 0xFF, v & 0xFF];
List<int> _ascii(String s) => latin1.encode(s);

/// A big-endian TIFF header and IFD0 with Make, Model, Orientation and a GPS IFD holding a latitude.
List<int> _tiff({int? orientation}) {
  final make = [..._ascii(_make), 0];
  final model = [..._ascii(_model), 0];
  final entries = 3 + (orientation == null ? 0 : 1);
  final ifd0End = 8 + 2 + entries * 12 + 4;
  final makeAt = ifd0End;
  final modelAt = makeAt + make.length;
  final gpsAt = modelAt + model.length;
  List<int> entry(int tag, int type, int count, List<int> value) => [
    ..._u16(tag),
    ..._u16(type),
    ..._u32(count),
    ...value,
  ];
  final gpsEntries = 2;
  final latAt = gpsAt + 2 + gpsEntries * 12 + 4;
  return [
    ..._ascii('MM'), 0, 42, ..._u32(8),
    ..._u16(entries),
    ...entry(0x010F, 2, make.length, _u32(makeAt)),
    ...entry(0x0110, 2, model.length, _u32(modelAt)),
    if (orientation != null) ...entry(0x0112, 3, 1, [..._u16(orientation), 0, 0]),
    ...entry(0x8825, 4, 1, _u32(gpsAt)),
    ..._u32(0),
    ...make,
    ...model,
    ..._u16(gpsEntries),
    ...entry(0x0001, 2, 2, [0x4E, 0, 0, 0]), // N
    ...entry(0x0002, 5, 3, _u32(latAt)),
    ..._u32(0),
    ..._u32(43),
    ..._u32(1),
    ..._u32(51),
    ..._u32(1),
    ..._u32(2268),
    ..._u32(100),
  ];
}

List<int> _segment(int marker, List<int> payload) => [0xFF, marker, ..._u16(payload.length + 2), ...payload];

/// The image itself: a frame header, a scan header and entropy-coded bytes with a stuffed 0xFF and a restart
/// marker, as the stripper must pass them through untouched.
final _image = [
  ..._segment(0xDB, List.filled(65, 1)),
  ..._segment(0xC0, [8, 0, 16, 0, 16, 1, 1, 0x11, 0]),
  ..._segment(0xDA, [1, 1, 0, 0, 63, 0]),
  0x12,
  0xFF,
  0x00,
  0x34,
  0xFF,
  0xD0,
  0x56,
  0xFF,
  0xD9,
];

Uint8List _jpeg({int? orientation = 6, bool trailer = true}) => _bytes([
  0xFF, 0xD8,
  ..._segment(0xE0, [..._ascii('JFIF'), 0, 1, 1, 0, 0, 1, 0, 1, 0, 0]),
  ..._segment(0xE1, [..._ascii('Exif'), 0, 0, ..._tiff(orientation: orientation)]),
  ..._segment(0xE1, _ascii(_xmp)),
  ..._segment(0xE2, [..._ascii('ICC_PROFILE'), 0, 1, 1, 9, 9]),
  ..._segment(0xED, _ascii(_iptc)),
  ..._segment(0xFE, _ascii(_comment)),
  ..._image,
  // A second picture after the first one's end, as a phone appends its preview or depth map, with its own Exif.
  if (trailer) ...[
    0xFF,
    0xD8,
    ..._segment(0xE1, [..._ascii('Exif'), 0, 0, ..._tiff(orientation: 1)]),
    0xFF,
    0xD9,
  ],
]);

/// The APP segments of [jpeg] before its frame, by marker.
List<(int, Uint8List)> _apps(Uint8List jpeg) {
  final out = <(int, Uint8List)>[];
  var i = 2;
  while (jpeg[i] == 0xFF && jpeg[i + 1] >= 0xE0 && jpeg[i + 1] <= 0xFE) {
    final len = jpeg[i + 2] << 8 | jpeg[i + 3];
    out.add((jpeg[i + 1], Uint8List.sublistView(jpeg, i + 4, i + 2 + len)));
    i += 2 + len;
  }
  return out;
}

bool _has(Uint8List bytes, String s) {
  final needle = _ascii(s);
  outer:
  for (var i = 0; i + needle.length <= bytes.length; i++) {
    for (var k = 0; k < needle.length; k++) {
      if (bytes[i + k] != needle[k]) continue outer;
    }
    return true;
  }
  return false;
}

Uint8List _chunk(String type, List<int> data) {
  final body = [..._ascii(type), ...data];
  return _bytes([..._u32(data.length), ...body, ..._u32(crc32(body))]);
}

Uint8List _png({int? orientation = 8}) => _bytes([
  0x89,
  0x50,
  0x4E,
  0x47,
  0x0D,
  0x0A,
  0x1A,
  0x0A,
  ..._chunk('IHDR', [..._u32(1), ..._u32(1), 8, 2, 0, 0, 0]),
  ..._chunk('sRGB', [0]),
  ..._chunk('tEXt', _ascii('Author\u0000$_make')),
  ..._chunk('iTXt', _ascii('XML:com.adobe.xmp\u0000\u0000\u0000\u0000\u0000$_xmp')),
  ..._chunk('tIME', [7, 234, 11, 4, 10, 0, 0]),
  ..._chunk('eXIf', _tiff(orientation: orientation)),
  ..._chunk('IDAT', [0x78, 0x9C, 0x63, 0x60, 0x00, 0x00, 0x00, 0x02, 0x00, 0x01]),
  ..._chunk('IEND', []),
]);

List<String> _chunkTypes(Uint8List png) {
  final types = <String>[];
  for (var i = 8; i < png.length;) {
    final len = ByteData.sublistView(png).getUint32(i);
    types.add(latin1.decode(png.sublist(i + 4, i + 8)));
    i += 12 + len;
  }
  return types;
}

List<int> _box(String type, List<int> payload) => [..._u32(payload.length + 8), ..._ascii(type), ...payload];

/// A full box with version 0 creation and modification times.
List<int> _timed(String type, List<int> rest) =>
    _box(type, [0, 0, 0, 0, ..._u32(0x7A000001), ..._u32(0x7A000002), ...rest]);

/// A 90° rotation matrix, the orientation of a portrait phone video.
final _matrix = [
  ..._u32(0),
  ..._u32(0x00010000),
  ..._u32(0),
  ..._u32(0xFFFF0000),
  ..._u32(0),
  ..._u32(0),
  ..._u32(0),
  ..._u32(0),
  ..._u32(0x40000000),
];

List<int> _trak(String handler, String entry) => _box('trak', [
  ..._timed('tkhd', [..._u32(1), ..._u32(0), ..._u32(1000), ...List.filled(16, 0), ..._matrix, ..._u32(0), ..._u32(0)]),
  ..._box('mdia', [
    ..._timed('mdhd', [..._u32(1000), ..._u32(1000), 0, 0, 0, 0]),
    ..._box('hdlr', [0, 0, 0, 0, 0, 0, 0, 0, ..._ascii(handler), ...List.filled(12, 0), 0]),
    ..._box('minf', [
      ..._box('stbl', [
        ..._box('stsd', [0, 0, 0, 0, ..._u32(1), ..._box(entry, List.filled(8, 0))]),
      ]),
    ]),
  ]),
  ..._box('udta', _box('©mak', _ascii(_make))),
]);

/// A movie with `moov` first or last, H.264 video and AAC audio unless told otherwise, and the metadata a phone
/// writes: location and device in `udta`, Apple's keys in `meta`, XMP in a `uuid` box, creation times.
Uint8List _movie({String brand = 'isom', String video = 'avc1', String? audio = 'mp4a', bool moovFirst = true}) {
  final moov = _box('moov', [
    ..._timed('mvhd', [..._u32(1000), ..._u32(1000), ...List.filled(80, 0)]),
    ..._trak('vide', video),
    if (audio != null) ..._trak('soun', audio),
    ..._box('udta', [
      ..._box('©xyz', [0, 18, 0, 0, ..._ascii(_location)]),
      ..._box('©mod', _ascii(_model)),
    ]),
    ..._box('meta', [0, 0, 0, 0, ..._box('keys', _ascii('com.apple.quicktime.location.ISO6709 $_location'))]),
  ]);
  final mdat = _box('mdat', List.generate(4096, (i) => i % 251));
  return _bytes([
    ..._box('ftyp', [..._ascii(brand), ..._u32(0), ..._ascii(brand), ..._ascii('mp41')]),
    if (moovFirst) ...moov,
    ...mdat,
    if (!moovFirst) ...moov,
    ..._box('uuid', [...List.filled(16, 0xBE), ..._ascii('<x:xmpmeta>$_location</x:xmpmeta>')]),
  ]);
}

/// The payload of the first box named by [path] (`moov/trak/tkhd`).
Uint8List? _find(Uint8List b, List<String> path, [int start = 0, int? end]) {
  end ??= b.length;
  for (var i = start; i + 8 <= end;) {
    final size = ByteData.sublistView(b).getUint32(i);
    final type = latin1.decode(b.sublist(i + 4, i + 8));
    if (type == path.first) {
      return path.length == 1 ? Uint8List.sublistView(b, i + 8, i + size) : _find(b, path.sublist(1), i + 8, i + size);
    }
    i += size;
  }
  return null;
}

void main() {
  group('detect', () {
    for (final (name, head, format) in [
      ('JPEG', _jpeg().sublist(0, 16), Format.jpeg),
      ('PNG', _png().sublist(0, 16), Format.png),
      ('PDF', _ascii('%PDF-1.7\n'), Format.pdf),
      ('PDF after a byte order mark', [0xEF, 0xBB, 0xBF, ..._ascii('%PDF-1.4')], Format.pdf),
      ('MP4', _movie().sublist(0, 32), Format.mp4),
      ('MOV', _movie(brand: 'qt  ').sublist(0, 32), Format.mov),
      ('HEIC', _box('ftyp', [..._ascii('heic'), ..._u32(0), ..._ascii('mif1heic')]), Format.heif),
      ('HEIF', _box('ftyp', [..._ascii('mif1'), ..._u32(0), ..._ascii('mif1')]), Format.heif),
      ('text', _ascii('plain text called photo.jpg'), Format.unknown),
      ('nothing', <int>[], Format.unknown),
    ]) {
      test(name, () => expect(detect(head), format));
    }
  });

  group('a photo is stripped of its metadata and keeps its orientation', () {
    test('JPEG: GPS, device, XMP, IPTC, comments and appended pictures removed; only the orientation is left', () {
      final original = _jpeg();
      final stripped = stripJpeg(original);

      for (final s in [_make, _model, 'synthetic location', 'synthetic caption', _comment, 'JFIF']) {
        expect(_has(stripped, s), isFalse, reason: s);
      }
      final apps = _apps(stripped);
      expect([for (final (m, _) in apps) m], [0xE1, 0xE2], reason: 'one Exif with the orientation, the colour profile');
      expect(apps.first.$2, [
        ..._ascii('Exif'),
        0,
        0,
        ..._ascii('MM'),
        0,
        42,
        ..._u32(8),
        ..._u16(1),
        ..._u16(0x0112),
        ..._u16(3),
        ..._u32(1),
        ..._u16(6),
        0,
        0,
        ..._u32(0),
      ], reason: 'IFD0 holds Orientation 6 and nothing else: no GPS IFD');
      expect(
        stripped.sublist(stripped.length - _image.length),
        _image,
        reason: 'the picture itself untouched, then the end',
      );
    });

    test('JPEG without an orientation keeps none; an upright one needs none', () {
      for (final o in [null, 1]) {
        final stripped = stripJpeg(_jpeg(orientation: o, trailer: false));
        expect([for (final (m, _) in _apps(stripped)) m], [0xE2], reason: '$o');
      }
    });

    test('JPEG that is not one is refused', () {
      expect(() => stripJpeg(_bytes(_ascii('not a JPEG'))), throwsFormatException);
      expect(() => stripJpeg(_jpeg().sublist(0, 40)), throwsFormatException, reason: 'cut short');
    });

    test('PNG: text, XMP and time removed; eXIf reduced to the orientation', () {
      final stripped = stripPng(_png());

      expect(_chunkTypes(stripped), ['IHDR', 'eXIf', 'sRGB', 'IDAT', 'IEND']);
      for (final s in [_make, _model, 'synthetic location']) {
        expect(_has(stripped, s), isFalse, reason: s);
      }
      expect(_chunkTypes(stripPng(_png(orientation: null))), ['IHDR', 'sRGB', 'IDAT', 'IEND']);
    });

    test('PNG chunks keep valid checksums', () {
      final stripped = stripPng(_png());
      final data = ByteData.sublistView(stripped);
      for (var i = 8; i < stripped.length;) {
        final len = data.getUint32(i);
        expect(data.getUint32(i + 8 + len), crc32(stripped.sublist(i + 4, i + 8 + len)));
        i += 12 + len;
      }
    });
  });

  group('a video', () {
    late Directory dir;
    setUp(() => dir = Directory.systemTemp.createTempSync('salus-prep-'));
    tearDown(() => dir.deleteSync(recursive: true));

    File write(String name, List<int> bytes) => File('${dir.path}/$name')..writeAsBytesSync(bytes);

    for (final first in [true, false]) {
      test(
        'is stripped of location, device, keys, XMP and times, keeps its rotation (moov ${first ? 'first' : 'last'})',
        () async {
          final original = _movie(moovFirst: first);
          final out = File('${dir.path}/out.mp4');
          final movie = await readMovie(write('in.mp4', original));
          await writePatched(write('in.mp4', original), out, movie.patches);
          final stripped = out.readAsBytesSync();

          expect(stripped.length, original.length, reason: 'boxes become free space; no offset moves');
          for (final s in [_location, _make, _model, 'com.apple.quicktime']) {
            expect(_has(stripped, s), isFalse, reason: s);
          }
          expect(_find(stripped, ['moov', 'udta']), isNull);
          expect(_find(stripped, ['moov', 'meta']), isNull);
          expect(_find(stripped, ['uuid']), isNull);
          expect(
            _find(stripped, ['moov', 'mvhd'])!.sublist(4, 12),
            List.filled(8, 0),
            reason: 'creation and modification',
          );
          final tkhd = _find(stripped, ['moov', 'trak', 'tkhd'])!;
          expect(tkhd.sublist(4, 12), List.filled(8, 0));
          expect(tkhd.sublist(40, 76), _matrix, reason: 'the orientation');
          expect(_find(stripped, ['moov', 'trak', 'mdia', 'mdhd'])!.sublist(4, 12), List.filled(8, 0));
          expect(_find(stripped, ['mdat']), _find(original, ['mdat']), reason: 'the picture and sound untouched');
        },
      );
    }

    for (final (name, movie, plays) in [
      ('H.264 and AAC MP4', _movie(), true),
      ('H.264 without sound', _movie(audio: null), true),
      ('a MOV', _movie(brand: 'qt  '), false),
      ('HEVC', _movie(video: 'hvc1'), false),
      ('Opus sound', _movie(audio: 'Opus'), false),
    ]) {
      test('$name ${plays ? 'plays in Chrome as it is' : 'is converted'}', () async {
        expect((await readMovie(write('m.mp4', movie))).playsInChrome, plays);
      });
    }

    test('a box size that lies is refused', () async {
      final broken = _movie()..setRange(32, 36, _u32(0x7FFFFFFF));
      await expectLater(readMovie(write('m.mp4', broken)), throwsFormatException);
    });
  });

  group('prepare', () {
    late Directory dir;
    late List<String> converted;
    setUp(() {
      dir = Directory.systemTemp.createTempSync('salus-prep-');
      converted = [];
    });
    tearDown(() => dir.deleteSync(recursive: true));

    File write(String name, List<int> bytes) => File('${dir.path}/$name')..writeAsBytesSync(bytes);

    /// The platform's conversion: a HEIC becomes a JPEG still carrying metadata, a video an H.264/AAC MP4 still
    /// carrying its location, so the stripping after it is what removes them.
    Future<String> convert(Conversion c, String path) async {
      converted.add('${c.name} ${path.split('/').last}');
      return write(
        c == Conversion.photo ? 'converted.jpg' : 'converted.mp4',
        c == Conversion.photo ? _jpeg() : _movie(),
      ).path;
    }

    Future<Prepared> run(String name, List<int> bytes) => prepareFile(write(name, bytes).path, name, convert, dir);

    test('a JPEG is stripped, not converted', () async {
      final p = await run('IMG_0001.JPG', _jpeg());
      expect(converted, isEmpty);
      expect((p.kind, p.name), (FileKind.photo, 'IMG_0001.JPG'));
      final bytes = Uint8List.fromList(await p.read().expand((c) => c).toList());
      expect(_has(bytes, _make), isFalse);
      expect(p.size, bytes.length);
    });

    test('a HEIC is converted to JPEG, then stripped, and named .jpg', () async {
      final p = await run('IMG_0002.HEIC', _box('ftyp', [..._ascii('heic'), ..._u32(0), ..._ascii('mif1heic')]));
      expect(converted, ['photo IMG_0002.HEIC']);
      expect((p.kind, p.name), (FileKind.photo, 'IMG_0002.jpg'));
      expect(_has(Uint8List.fromList(await p.read().expand((c) => c).toList()), _model), isFalse);
    });

    test('a MOV is converted to MP4 and stripped; an H.264/AAC MP4 is only stripped', () async {
      final mov = await run('clip.MOV', _movie(brand: 'qt  '));
      expect(converted, ['video clip.MOV']);
      expect((mov.kind, mov.name), (FileKind.video, 'clip.mp4'));
      expect(_has(Uint8List.fromList(await mov.read().expand((c) => c).toList()), _location), isFalse);

      converted.clear();
      final mp4 = await run('clip.mp4', _movie());
      expect(converted, isEmpty);
      expect(_has(Uint8List.fromList(await mp4.read().expand((c) => c).toList()), _location), isFalse);
    });

    test('a PDF goes as it is', () async {
      final pdf = _ascii('%PDF-1.7\nsynthetic\n%%EOF');
      final p = await run('letter.pdf', pdf);
      expect((p.kind, p.size), (FileKind.document, pdf.length));
      expect(await p.read().expand((c) => c).toList(), pdf);
    });

    test('the copies it made are deleted when it is discarded; the picked file is not', () async {
      final picked = write('IMG_0003.JPG', _jpeg());
      final p = await prepareFile(picked.path, 'IMG_0003.JPG', convert, dir);
      final made = dir.listSync().whereType<File>().map((f) => f.path).toSet()..remove(picked.path);
      expect(made, hasLength(1));
      await p.discard();
      expect(made.where((f) => File(f).existsSync()), isEmpty);
      expect(picked.existsSync(), isTrue);
    });

    test('anything else, a failed conversion and a video still not H.264 after it cannot be prepared', () async {
      await expectLater(run('notes.txt', _ascii('plain text')), throwsA(isA<CannotPrepare>()));
      Future<String> fails(Conversion c, String path) async => throw Exception('synthetic');
      await expectLater(
        prepareFile(write('x.HEIC', _box('ftyp', [..._ascii('heic'), ..._u32(0)])).path, 'x.HEIC', fails, dir),
        throwsA(isA<CannotPrepare>()),
      );
      Future<String> stillHevc(Conversion c, String path) async => write('h.mp4', _movie(video: 'hvc1')).path;
      await expectLater(
        prepareFile(write('x.mov', _movie(brand: 'qt  ')).path, 'x.mov', stillHevc, dir),
        throwsA(isA<CannotPrepare>()),
      );
    });

    test('a photo past 20 MB is too large before it is read', () async {
      final big = write('big.jpg', _jpeg());
      final raf = big.openSync(mode: FileMode.append)..setPositionSync(20 * 1000 * 1000);
      raf
        ..writeByteSync(0)
        ..closeSync();
      await expectLater(prepareFile(big.path, 'big.jpg', convert, dir), throwsA(isA<TooLarge>()));
    });
  });
}
