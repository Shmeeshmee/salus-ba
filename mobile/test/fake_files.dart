import 'dart:async';
import 'dart:typed_data';

import 'package:salus/file_prep.dart';
import 'package:salus/files.dart';

/// The camera, photo library and document picker as the questionnaire screen sees them: what the next pick
/// answers, and the prepared file's bytes, which a test can hold back chunk by chunk to watch the progress.
class FakeFiles implements Files {
  /// Every pick, with what the library was asked for.
  final picks = <(Source, bool, bool)>[];

  /// The next pick; null when the patient cancels.
  Picked? next = const Picked(path: '/synthetic/cache/IMG_0001.HEIC', name: 'IMG_0001.HEIC');
  Object? pickError;

  /// Holds the preparation until the test completes it.
  Completer<void>? preparing;
  Object? prepareError;
  var name = 'synthetic-photo.jpg';
  var kind = FileKind.photo;

  /// The prepared file's bytes, in chunks; while [gate] is set each chunk waits for its completer.
  var chunks = <List<int>>[List.filled(1000, 1), List.filled(1000, 2), List.filled(1000, 3)];
  List<Completer<void>>? gate;
  var discarded = 0;

  int get size => chunks.fold(0, (n, c) => n + c.length);

  /// Every PDF handed to the share sheet, by name, and every text.
  final shared = <(String, Uint8List)>[];
  final sharedText = <String>[];

  @override
  Future<void> sharePdf(String name, Uint8List bytes) async => shared.add((name, bytes));

  @override
  Future<void> shareText(String text) async => sharedText.add(text);

  @override
  Future<Picked?> pick(Source source, {required bool photos, required bool videos}) async {
    picks.add((source, photos, videos));
    if (pickError != null) throw pickError!;
    return next;
  }

  @override
  Future<Prepared> prepare(Picked picked) async {
    await preparing?.future;
    if (prepareError != null) throw prepareError!;
    return Prepared(name: name, size: size, kind: kind, read: _read, discard: () async => discarded++);
  }

  Stream<List<int>> _read() async* {
    for (var i = 0; i < chunks.length; i++) {
      if (gate != null) await gate![i].future;
      yield chunks[i];
    }
  }
}
