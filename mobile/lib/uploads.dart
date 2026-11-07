import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

import 'booking.dart';
import 'file_prep.dart';
import 'files.dart';
import 'messages.dart';
import 'questionnaire.dart';
import 'session.dart';

// The files of one questionnaire response while the patient fills it in (docs/attachments/SPEC.md §2 "Mobile"):
// each picked file prepared, uploaded one at a time with its progress, then its status read until the scan says
// `clean` or `rejected`. Nothing here logs; a file's name is shown and nowhere else.

/// Where a file stands (COPY M4, M5).
enum FileState { preparing, uploading, checking, added, failed }

class FileEntry {
  FileEntry._(this.linkId, this.title, this.state, {this.id, this.kind, this.size = 0, this.contentType = ''});

  final String linkId;

  /// The binary's id once the server has the file.
  String? id;
  String title;
  FileKind? kind;

  /// The type the server detected, once it has the file.
  String contentType;
  int size;
  FileState state;
  int percent = 0;

  /// Kept until the upload succeeds, for "Try again".
  Prepared? _file;
  var _polls = 0;
  Timer? _poll;

  Attached? get attached => id == null ? null : Attached(id: id!, title: title, contentType: contentType, size: size);
}

/// How long before the [n]th status read (from 0): every 2 s for the first 10 s, then 2 s longer each time up to
/// every 10 s.
Duration pollDelay(int n) => Duration(seconds: n < 5 ? 2 : math.min(10, 2 * (n - 3)));

/// The response-level refusals of an upload, as for the answers (questionnaire M16): they end filling in.
const _ending = {'submitted', 'withdrawn', 'past_due', 'consent_required', 'appointment_closed', 'not_found'};

class Uploads extends ChangeNotifier {
  /// [attached] are the files the response already has, by item; [onEnded] hears a refusal that ends filling in.
  Uploads({
    required this.booking,
    required this.files,
    required this.responseId,
    required this.items,
    required Map<String, List<Attached>> attached,
    required this.onEnded,
  }) {
    for (final MapEntry(key: linkId, value: list) in attached.entries) {
      for (final a in list) {
        final e = FileEntry._(
          linkId,
          a.title,
          FileState.checking,
          id: a.id,
          kind: a.kind,
          size: a.size,
          contentType: a.contentType,
        );
        _entries.add(e);
        unawaited(_read(e));
      }
    }
  }

  final Booking booking;
  final Files files;
  final String responseId;
  final Map<String, Item> items;
  final void Function(ApiFailure failure) onEnded;

  final _entries = <FileEntry>[];
  final _errors = <String, String>{};
  final _queue = <FileEntry>[];
  FileEntry? _current;
  Completer<void>? _abort;
  var _disposed = false;

  List<FileEntry> of(String linkId) => [
    for (final e in _entries)
      if (e.linkId == linkId) e,
  ];

  /// The last problem of an item's files (M7, M8, M9).
  String? error(String linkId) => _errors[linkId];

  /// A file not yet clean: Submit waits (M10).
  bool get pending => _entries.any((e) => e.state != FileState.added && e.state != FileState.failed);

  /// A file being prepared or sent: leaving stops it (M11).
  bool get uploading => _entries.any((e) => e.state == FileState.preparing || e.state == FileState.uploading);

  /// The files the server has, as the answers of their items.
  Answers get answers => {
    for (final linkId in {for (final e in _entries) e.linkId})
      if (of(linkId).map((e) => e.attached).nonNulls.toList() case final files when files.isNotEmpty) linkId: files,
  };

  /// Prepares and uploads [picked] to the item [linkId].
  Future<void> add(String linkId, Picked picked) async {
    final e = FileEntry._(linkId, picked.title ?? picked.name, FileState.preparing);
    _entries.add(e);
    _errors.remove(linkId);
    _changed();
    Prepared file;
    try {
      file = await files.prepare(picked);
    } on TooLarge catch (t) {
      return _drop(e, fileTooLarge(t.kind.maxMegabytes));
    } on Object {
      return _drop(e, cannotPrepareFile);
    }
    if (_disposed) return file.discard();
    if (picked.title case final title?) {
      file = Prepared(name: title, size: file.size, kind: file.kind, read: file.read, discard: file.discard);
    }
    e
      ..title = file.name
      ..kind = file.kind
      ..size = file.size
      .._file = file;
    _send(e);
  }

  /// Sends a failed file again, whole (SPEC §3 O3).
  void retry(FileEntry e) => _send(e);

  /// Removes a file: one the server has is deleted there first. Throws [ApiFailure] when that is refused.
  Future<void> remove(FileEntry e) async {
    final id = e.id;
    if (id != null) {
      try {
        await booking.removeFile(id);
      } on ApiFailure catch (f) {
        if (f.status != 404) rethrow;
      }
    }
    await _forget(e);
    _changed();
  }

  void _send(FileEntry e) {
    e
      ..state = FileState.uploading
      ..percent = 0;
    _queue.add(e);
    _changed();
    unawaited(_next());
  }

  Future<void> _next() async {
    if (_current != null || _queue.isEmpty || _disposed) return;
    final e = _current = _queue.removeAt(0);
    _abort = Completer();
    try {
      final b = await booking.upload(
        responseId,
        e.linkId,
        e._file!,
        onSent: (sent) => _progress(e, sent),
        abort: _abort!.future,
      );
      if (_disposed) return;
      e
        ..id = b.id
        ..contentType = b.contentType.toJson();
      await e._file?.discard();
      e._file = null;
      _status(e, b.status.toJson());
    } on http.RequestAbortedException {
      return;
    } on ApiFailure catch (f) {
      if (_disposed) return;
      await _refused(e, f);
    } finally {
      if (identical(_current, e)) _current = null;
    }
    unawaited(_next());
  }

  void _progress(FileEntry e, int sent) {
    final percent = e.size == 0 ? 100 : sent * 100 ~/ e.size;
    if (percent == e.percent) return;
    e.percent = percent;
    _changed();
  }

  Future<void> _refused(FileEntry e, ApiFailure f) async {
    if (f.unreachable || f.status >= 500) {
      e.state = FileState.failed;
      return _changed();
    }
    if (_ending.contains(f.code) || f.status == 404) {
      await _forget(e);
      _changed();
      return onEnded(f);
    }
    final item = items[e.linkId];
    await _drop(e, switch (f.code) {
      'too_large' => fileTooLarge((e.kind ?? FileKind.photo).maxMegabytes),
      'unsupported_type' => fileTypeRefused,
      'too_many_files' => tooManyFiles(item?.maxFiles ?? 1),
      'response_full' => filesFull,
      _ => fileNotUploaded,
    });
  }

  /// The file's status from the server: clean is added, rejected is gone (M7), quarantined is read again.
  void _status(FileEntry e, String status) {
    switch (status) {
      case 'clean':
        e.state = FileState.added;
      case 'rejected':
        unawaited(_drop(e, fileRejected));
        return;
      default:
        e.state = FileState.checking;
        e._poll = Timer(pollDelay(e._polls++), () => _read(e));
    }
    _changed();
  }

  Future<void> _read(FileEntry e) async {
    try {
      final b = await booking.fileStatus(e.id!);
      if (_disposed || !_entries.contains(e)) return;
      _status(e, b.status.toJson());
    } on ApiFailure catch (f) {
      if (_disposed || !_entries.contains(e)) return;
      if (f.status == 404) {
        await _forget(e);
        return _changed();
      }
      e._poll = Timer(pollDelay(e._polls++), () => _read(e));
    }
  }

  Future<void> _drop(FileEntry e, String message) async {
    await _forget(e);
    _errors[e.linkId] = message;
    _changed();
  }

  Future<void> _forget(FileEntry e) async {
    _entries.remove(e);
    _queue.remove(e);
    e._poll?.cancel();
    await e._file?.discard();
    e._file = null;
  }

  void _changed() {
    if (!_disposed) notifyListeners();
  }

  /// Stops the upload under way and every status read, and deletes the copies of files not sent.
  @override
  void dispose() {
    _disposed = true;
    if (_abort?.isCompleted == false) _abort!.complete();
    for (final e in [..._entries]) {
      e._poll?.cancel();
      unawaited(e._file?.discard());
      e._file = null;
    }
    super.dispose();
  }
}
