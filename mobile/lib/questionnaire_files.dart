import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

import 'call.dart' show Access, Kind, Media;
import 'file_prep.dart' show CannotPrepare, FileKind;
import 'files.dart';
import 'messages.dart';
import 'questionnaire.dart';
import 'uploads.dart';
import 'zones.dart';

// A file upload item on the questionnaire screen (docs/attachments/COPY.md M1 to M14): its hint, its files with
// their status, "Add file" and the sheet of sources the item accepts, and the files read only once submitted.

/// The hint under a file upload item (M2): a sentence per accepted kind, in this order, then the number of files.
String fileHint(Item item) => [
  if (item.accept.contains(FileKind.photo)) photosUpTo,
  if (item.accept.contains(FileKind.document)) documentsUpTo,
  if (item.accept.contains(FileKind.video)) videosUpTo,
  if (item.maxFiles == 1) oneFileHint else upToFilesHint(item.maxFiles),
].join(' ');

/// A size as M13 writes it: KB under 1 MB, MB with one decimal from there.
String fileSize(int bytes) {
  if (bytes < 999500) return kilobytes(bytes < 1000 ? 1 : (bytes / 1000).round());
  return megabytes((bytes / 1000000).toStringAsFixed(1));
}

String kindName(FileKind? kind) => switch (kind) {
  FileKind.photo => photoKind,
  FileKind.document => documentKind,
  FileKind.video => videoKind,
  null => '',
};

/// The number of files of a submitted answer (M12).
String fileCount(int n) => n == 1 ? oneFile : filesCount(n);

/// The sources [item] accepts, in M3's order.
List<(Source, String)> sourcesOf(Item item) {
  final photo = item.accept.contains(FileKind.photo);
  final video = item.accept.contains(FileKind.video);
  return [
    if (photo) (Source.camera, takePhoto),
    if (video) (Source.videoCamera, recordVideo),
    if (photo || video) (Source.library, chooseFromPhotos),
    if (item.accept.contains(FileKind.document)) (Source.document, chooseDocument),
  ];
}

/// The sheet after "Add file" (M3): the item's sources and "Cancel".
Future<Source?> chooseSource(BuildContext context, Item item) => showModalBottomSheet<Source>(
  context: context,
  builder: (context) => SafeArea(
    child: Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        for (final (source, label) in sourcesOf(item))
          ListTile(title: Text(label), onTap: () => Navigator.of(context).pop(source)),
        ListTile(title: const Text(cancel), onTap: () => Navigator.of(context).pop()),
      ],
    ),
  ),
);

/// The title of a file from the camera (M21, M22): its kind and the patient's local date and time at [taken].
String cameraTitle(Source source, DateTime taken) {
  final local = taken.toLocal();
  final date = Zone.longDate(DateTime.utc(local.year, local.month, local.day));
  final time = Zone.clock(local);
  return source == Source.videoCamera ? cameraVideoTitle(date, time) : cameraPhotoTitle(date, time);
}

/// "Add file": the sheet, the camera's permission when a camera source is chosen, then the picker. Answers the
/// picked file, a camera's titled by [now] when it came back (M21, M22), or the item's error (M9), or null when the
/// patient stopped.
Future<({Picked? picked, String? error})?> pickFile(
  BuildContext context,
  Item item, {
  required Media media,
  required Files files,
  required DateTime Function() now,
}) async {
  final source = await chooseSource(context, item);
  if (source == null || !context.mounted) return null;
  final camera = source == Source.camera || source == Source.videoCamera;
  if (camera && !await _cameraAllowed(context, media)) return null;
  try {
    final picked = await files.pick(
      source,
      photos: item.accept.contains(FileKind.photo),
      videos: item.accept.contains(FileKind.video),
    );
    if (picked == null) return null;
    final titled = camera ? Picked(path: picked.path, name: picked.name, title: cameraTitle(source, now())) : picked;
    return (picked: titled, error: null);
  } on AccessRefused catch (r) {
    if (context.mounted) {
      await _refused(context, media, r.refused == Refused.camera ? cameraDeniedForFiles : photosDenied);
    }
    return null;
  } on CannotPrepare {
    return (picked: null, error: cannotPrepareFile);
  }
}

/// Whether the camera may be used: on Android the app explains first (M18), then the system asks; refused, M19.
Future<bool> _cameraAllowed(BuildContext context, Media media) async {
  var access = await media.access(Kind.camera);
  if (access == Access.unasked && context.mounted) {
    if (defaultTargetPlatform == TargetPlatform.android && await _explainCamera(context) != true) return false;
    access = await media.ask(Kind.camera);
  }
  if (access == Access.allowed) return true;
  if (context.mounted) await _refused(context, media, cameraDeniedForFiles);
  return false;
}

Future<bool?> _explainCamera(BuildContext context) => showDialog<bool>(
  context: context,
  builder: (context) => AlertDialog(
    title: const Text(allowCamera),
    content: const Text(whyCameraForFiles),
    actions: [
      TextButton(onPressed: () => Navigator.of(context).pop(false), child: const Text(notNow)),
      TextButton(onPressed: () => Navigator.of(context).pop(true), child: const Text(continueLabel)),
    ],
  ),
);

/// M19 or M20: access refused, with the way to the device settings.
Future<void> _refused(BuildContext context, Media media, String text) => showDialog<void>(
  context: context,
  builder: (context) => AlertDialog(
    content: Text(text),
    actions: [
      TextButton(onPressed: () => Navigator.of(context).pop(), child: const Text(cancel)),
      TextButton(
        onPressed: () {
          Navigator.of(context).pop();
          media.openSettings();
        },
        child: const Text(openSettings),
      ),
    ],
  ),
);

/// A file upload item while it is filled in.
class FileField extends StatelessWidget {
  const FileField({
    super.key,
    required this.item,
    required this.header,
    required this.files,
    required this.enabled,
    required this.onAdd,
    required this.onRetry,
    required this.onRemove,
    this.error,
  });

  final Item item;
  final Widget header;
  final List<FileEntry> files;
  final bool enabled;
  final VoidCallback onAdd;
  final ValueChanged<FileEntry> onRetry;
  final ValueChanged<FileEntry> onRemove;
  final String? error;

  @override
  Widget build(BuildContext context) {
    final small = Theme.of(context).textTheme.bodySmall;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        header,
        Text(fileHint(item), style: small),
        for (final f in files)
          _FileRow(file: f, enabled: enabled, onRetry: () => onRetry(f), onRemove: () => onRemove(f)),
        if (enabled && files.length < item.maxFiles)
          Align(
            alignment: Alignment.centerLeft,
            child: OutlinedButton(onPressed: onAdd, child: const Text(addFile)),
          ),
        if (error != null) Text(error!, style: TextStyle(color: Theme.of(context).colorScheme.error)),
      ],
    );
  }
}

class _FileRow extends StatelessWidget {
  const _FileRow({required this.file, required this.enabled, required this.onRetry, required this.onRemove});

  final FileEntry file;
  final bool enabled;
  final VoidCallback onRetry;
  final VoidCallback onRemove;

  String get _status => switch (file.state) {
    FileState.preparing => preparing,
    FileState.uploading => uploadingPercent(file.percent),
    FileState.checking => checkingFile,
    FileState.added => added,
    FileState.failed => notUploaded,
  };

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final f = file;
    final kind = kindName(f.kind);
    final failed = f.state == FileState.failed;
    final removable = enabled && (f.state == FileState.checking || f.state == FileState.added);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              Expanded(
                // Its own node for TalkBack and VoiceOver, not merged into the question (M14).
                child: Semantics(
                  container: true,
                  label: fileLabel(kind, f.title, _status),
                  excludeSemantics: true,
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(f.title),
                      if (f.kind != null) Text(kindAndSize(kind, fileSize(f.size)), style: theme.textTheme.bodySmall),
                      Text(
                        _status,
                        style: failed ? TextStyle(color: theme.colorScheme.error) : theme.textTheme.bodySmall,
                      ),
                    ],
                  ),
                ),
              ),
              if (removable) IconButton(icon: const Icon(Icons.close), tooltip: removeFile, onPressed: onRemove),
            ],
          ),
          if (f.state == FileState.uploading)
            Semantics(
              container: true,
              child: LinearProgressIndicator(
                value: f.percent / 100,
                semanticsLabel: uploadingLabel(f.title, f.percent),
              ),
            ),
          if (failed && enabled)
            Row(
              children: [
                TextButton(onPressed: onRetry, child: const Text(tryAgain)),
                TextButton(onPressed: onRemove, child: const Text(remove)),
              ],
            ),
        ],
      ),
    );
  }
}

/// A submitted file upload answer read only (M12, M13): the number of files, then each title with its kind and size.
List<Widget> filesRead(BuildContext context, List<Attached> files) => [
  Text(fileCount(files.length)),
  for (final f in files) ...[
    Text(f.title),
    Text(kindAndSize(kindName(f.kind), fileSize(f.size)), style: Theme.of(context).textTheme.bodySmall),
  ],
];
