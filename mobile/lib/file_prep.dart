import 'dart:io';

import 'metadata.dart';

// A picked file made ready to upload (docs/attachments/SPEC.md §2 "Mobile", "Image and video metadata"): its type
// read from its bytes, a HEIC or HEIF photo converted to JPEG and a video that is not H.264/AAC MP4 converted, on
// the device, so that everything plays in Chrome; then its metadata removed (metadata.dart). Tested on generated
// files; only the conversion is platform code, passed in. Nothing here logs.

/// The kinds of file an attachment item accepts, with the server's limits (SPEC §3 O2).
enum FileKind {
  photo(20),
  document(20),
  video(200);

  const FileKind(this.maxMegabytes);

  final int maxMegabytes;
  int get maxBytes => maxMegabytes * 1000 * 1000;
}

/// What the platform converts: a photo to JPEG, a video to H.264/AAC MP4.
enum Conversion { photo, video }

/// Converts the file at a path on the device and answers the path of the result, in the temporary directory.
typedef Convert = Future<String> Function(Conversion conversion, String path);

/// The file could not be read, converted or understood (M9).
class CannotPrepare implements Exception {
  const CannotPrepare();
}

/// The file is larger than its kind allows (M8 `too_large`).
class TooLarge implements Exception {
  const TooLarge(this.kind);
  final FileKind kind;
}

/// A file ready to upload: its name (the title), size, kind and bytes. [discard] deletes the copies made for it.
class Prepared {
  const Prepared({
    required this.name,
    required this.size,
    required this.kind,
    required this.read,
    required this.discard,
  });

  final String name;
  final int size;
  final FileKind kind;
  final Stream<List<int>> Function() read;
  final Future<void> Function() discard;
}

/// Reads, converts where needed and strips the picked file at [path] named [name]; the copies go into [temp].
/// Throws [CannotPrepare] or [TooLarge].
Future<Prepared> prepareFile(String path, String name, Convert convert, Directory temp) async {
  final made = <File>[];
  Future<void> discard() async {
    for (final f in made) {
      if (f.existsSync()) await f.delete();
    }
  }

  try {
    return await _prepare(File(path), name, convert, temp, made, discard);
  } on TooLarge {
    await discard();
    rethrow;
  } on Object {
    await discard();
    throw const CannotPrepare();
  }
}

Future<Prepared> _prepare(
  File picked,
  String name,
  Convert convert,
  Directory temp,
  List<File> made,
  Future<void> Function() discard,
) async {
  Future<File> converted(Conversion c) async {
    final f = File(await convert(c, picked.path));
    made.add(f);
    return f;
  }

  Prepared ready(File f, String name, FileKind kind) {
    final size = f.lengthSync();
    if (size > kind.maxBytes) throw TooLarge(kind);
    return Prepared(name: name, size: size, kind: kind, read: f.openRead, discard: discard);
  }

  switch (detect(await _head(picked))) {
    case Format.jpeg || Format.png:
      return ready(await _strippedPhoto(picked, temp, made), name, FileKind.photo);
    case Format.heif:
      final jpeg = await converted(Conversion.photo);
      return ready(await _strippedPhoto(jpeg, temp, made), _renamed(name, 'jpg'), FileKind.photo);
    case Format.pdf:
      return ready(picked, name, FileKind.document);
    case Format.mp4 || Format.mov:
      var movie = picked;
      var info = await readMovie(movie);
      if (!info.playsInChrome) {
        movie = await converted(Conversion.video);
        info = await readMovie(movie);
        if (!info.playsInChrome) throw const CannotPrepare();
      }
      final out = _temporary(temp, 'mp4', made);
      await writePatched(movie, out, info.patches);
      return ready(out, movie == picked ? name : _renamed(name, 'mp4'), FileKind.video);
    case Format.unknown:
      throw const CannotPrepare();
  }
}

Future<List<int>> _head(File f) async {
  final raf = await f.open();
  try {
    return await raf.read(1024);
  } finally {
    await raf.close();
  }
}

/// The photo at [f] stripped into a new file; refused past the photo limit before it is read into memory.
Future<File> _strippedPhoto(File f, Directory temp, List<File> made) async {
  if (f.lengthSync() > FileKind.photo.maxBytes) throw const TooLarge(FileKind.photo);
  final bytes = await f.readAsBytes();
  final png = detect(bytes) == Format.png;
  final out = _temporary(temp, png ? 'png' : 'jpg', made);
  await out.writeAsBytes(png ? stripPng(bytes) : stripJpeg(bytes), flush: true);
  return out;
}

var _serial = 0;

File _temporary(Directory temp, String extension, List<File> made) {
  final f = File('${temp.path}/salus-upload-${DateTime.now().microsecondsSinceEpoch}-${_serial++}.$extension');
  made.add(f);
  return f;
}

String _renamed(String name, String extension) {
  final dot = name.lastIndexOf('.');
  return '${dot > 0 ? name.substring(0, dot) : name}.$extension';
}
