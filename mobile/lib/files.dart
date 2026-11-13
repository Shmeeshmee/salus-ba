import 'dart:io';

import 'package:file_picker/file_picker.dart' as fp;
import 'package:flutter/services.dart';
import 'package:image_picker/image_picker.dart';
import 'package:share_plus/share_plus.dart';

import 'copies.dart';
import 'file_prep.dart';

// files.dart is the one file that reaches the device's camera, photo library and document picker (image_picker,
// file_picker), the platform conversion (MainActivity.kt, AppDelegate.swift) and the share sheet (share_plus). The
// questionnaire and documents screens see only [Files], so they are tested with a fake (test/fake_files.dart); what
// is done to the bytes is in file_prep.dart and tested there. Nothing here logs.

/// Where a file comes from (COPY M3): the camera as a photo or a video, the photo library, or the documents.
enum Source { camera, videoCamera, library, document }

/// A file the patient picked: where the picker put it and its name. [title], when set, is sent and shown in place
/// of the prepared file's name: a camera's file is named by when it was taken, not by the picker's cache name.
class Picked {
  const Picked({required this.path, required this.name, this.title});

  final String path;
  final String name;
  final String? title;
}

/// What the patient refused the app: the camera (M19) or the photo library (M20).
enum Refused { camera, photos }

class AccessRefused implements Exception {
  const AccessRefused(this.refused);
  final Refused refused;
}

abstract interface class Files {
  /// A file from [source], or null when the patient cancelled. From the library, [photos] and [videos] say what
  /// may be chosen. Throws [AccessRefused], or [CannotPrepare] when the picker fails.
  Future<Picked?> pick(Source source, {required bool photos, required bool videos});

  /// The picked file converted and stripped of its metadata (file_prep.dart). Throws [CannotPrepare] or
  /// [TooLarge].
  Future<Prepared> prepare(Picked picked);

  /// Hands a signed document's PDF to the device's share sheet as [name]: to open it in a viewer, save it to the
  /// device's files or send it (docs/consultation/SPEC.md "Patient documents").
  Future<void> sharePdf(String name, Uint8List bytes);

  /// Hands [text] (a transcript) to the share sheet.
  Future<void> shareText(String text);

  /// Hands a file of the patient's answers to the share sheet as [name], in its [contentType]: to open it in a
  /// viewer, save it or send it (their history, SAL-39).
  Future<void> shareFile(String name, Uint8List bytes, String contentType);
}

/// [Files] on image_picker, file_picker and the app's own conversion channel. Made once, at start, when it sweeps
/// the copies an earlier run left (copies.dart); a pick waits for the sweep.
class DeviceFiles implements Files {
  DeviceFiles();

  final _images = ImagePicker();
  static const _channel = MethodChannel('ba.salus.salus/files');
  final Future<void> _swept = sweepCopies(Directory.systemTemp);

  @override
  Future<Picked?> pick(Source source, {required bool photos, required bool videos}) async {
    await _swept;
    try {
      if (source == Source.document) return await _document();
      // requestFullMetadata false: no photo library permission on iOS (the system picker runs outside the app),
      // and no metadata asked for that would be stripped anyway.
      final file = switch (source) {
        Source.camera => await _images.pickImage(source: ImageSource.camera, requestFullMetadata: false),
        Source.videoCamera => await _images.pickVideo(source: ImageSource.camera),
        _ when photos && videos => await _images.pickMedia(requestFullMetadata: false),
        _ when videos => await _images.pickVideo(source: ImageSource.gallery),
        _ => await _images.pickImage(source: ImageSource.gallery, requestFullMetadata: false),
      };
      return file == null ? null : Picked(path: file.path, name: file.name);
    } on PlatformException catch (e) {
      throw switch (e.code) {
        'camera_access_denied' => const AccessRefused(Refused.camera),
        'photo_access_denied' => const AccessRefused(Refused.photos),
        _ => const CannotPrepare(),
      };
    }
  }

  /// A PDF from the documents; copied into the temporary directory when the picker gives no local path.
  Future<Picked?> _document() async {
    final file = await fp.FilePicker.pickFile(type: fp.FileType.custom, allowedExtensions: const ['pdf']);
    if (file == null) return null;
    final path = file.path;
    if (path != null) return Picked(path: path, name: file.name);
    final copy = File('${Directory.systemTemp.path}/salus-picked-${DateTime.now().microsecondsSinceEpoch}.pdf');
    final sink = copy.openWrite();
    await sink.addStream(file.readAsByteStream());
    await sink.close();
    return Picked(path: copy.path, name: file.name);
  }

  @override
  Future<Prepared> prepare(Picked picked) async {
    final temp = Directory.systemTemp;
    // The pickers copy what they hand over into the app's temporary or cache directory; that copy is deleted with
    // the prepared file, wherever in those the picker put it. A file anywhere else is the patient's own and is never
    // touched (copies.dart).
    Future<void> forgetPicked() => forgetCopy(picked.path, temp);

    try {
      final p = await prepareFile(picked.path, picked.name, _convert, temp);
      return Prepared(
        name: p.name,
        size: p.size,
        kind: p.kind,
        read: p.read,
        discard: () async {
          await p.discard();
          await forgetPicked();
        },
      );
    } on Object {
      await forgetPicked();
      rethrow;
    }
  }

  /// The PDF goes through a copy in the temporary directory (`salus-document-*`), deleted once the sheet has closed,
  /// and swept at the next start if the app stopped first (copies.dart), as share_plus's own copy is.
  @override
  Future<void> sharePdf(String name, Uint8List bytes) => shareFile(name, bytes, 'application/pdf');

  @override
  Future<void> shareFile(String name, Uint8List bytes, String contentType) async {
    final temp = Directory.systemTemp;
    final copy = File('${temp.path}/salus-document-${DateTime.now().microsecondsSinceEpoch}');
    try {
      await copy.writeAsBytes(bytes, flush: true);
      await SharePlus.instance.share(
        ShareParams(
          files: [XFile(copy.path, mimeType: contentType)],
          fileNameOverrides: [name],
        ),
      );
    } finally {
      await forgetCopy(copy.path, temp);
    }
  }

  @override
  Future<void> shareText(String text) => SharePlus.instance.share(ShareParams(text: text));

  static Future<String> _convert(Conversion conversion, String path) async {
    final out = await _channel.invokeMethod<String>('convert', {'kind': conversion.name, 'path': path});
    if (out == null) throw const CannotPrepare();
    return out;
  }
}
