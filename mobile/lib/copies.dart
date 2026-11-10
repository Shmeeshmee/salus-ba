import 'dart:io';

// The copies of picked files on the device (docs/attachments/SPEC.md §2 "Mobile"). The pickers copy what the
// patient picks, with its metadata, into the app's own directories: image_picker and file_picker into the cache
// directory on Android (`<data>/cache`, `<data>/cache/<uuid>/`, `<data>/cache/file_picker/<stamp>/`), while
// Directory.systemTemp is often `<data>/code_cache` beside it; on iOS into `tmp` (`image_picker_*`, `<UUID>/`).
// The conversion writes `salus-converted-*` and file_prep.dart `salus-upload-*`. A copy is deleted once its upload
// ends, and what an earlier run left (the app stopped mid-upload) is swept at start. A signed document shared from
// the app goes through `salus-document-*`, and share_plus copies it into `<cache>/share_plus/` on Android, swept too. A file outside these
// directories is the patient's own and is never touched. path_provider is not a dependency of the app, so the
// directories are derived from Directory.systemTemp. Nothing here logs.

/// The directories the copies go into: [temp] (Directory.systemTemp) and the cache directory beside it, `cache` on
/// Android and `Library/Caches` on iOS.
List<String> _appDirectories(Directory temp) {
  final t = _real(temp.absolute.path);
  final parent = File(t).parent.path;
  return {t, '$parent/cache', '$parent/Library/Caches'}.toList();
}

/// The path with its symbolic links resolved where it exists (Android's `/data/user/0` is a link).
String _real(String path) {
  try {
    return File(path).resolveSymbolicLinksSync();
  } on FileSystemException {
    return path;
  }
}

/// The app directory [path] lies in, or null. A path with a `..` segment is in none.
String? _ownerOf(String path, Directory temp) {
  final absolute = File(path).absolute.path;
  if (absolute.split('/').contains('..')) return null;
  final real = '${_real(File(absolute).parent.path)}/${absolute.split('/').last}';
  for (final dir in _appDirectories(temp)) {
    if (real.startsWith('$dir/')) return dir;
  }
  return null;
}

/// Whether [path] is a copy in one of the app's own directories, not the patient's own file.
bool isAppCopy(String path, Directory temp) => _ownerOf(path, temp) != null;

/// Deletes [path] when it is a copy in the app's own directories, with the folders the picker made for it once
/// they are empty.
Future<void> forgetCopy(String path, Directory temp) async {
  final owner = _ownerOf(path, temp);
  if (owner == null) return;
  final f = File(path);
  try {
    if (f.existsSync()) await f.delete();
    for (var dir = f.parent; _real(dir.path).startsWith('$owner/'); dir = dir.parent) {
      if (dir.listSync().isNotEmpty) break;
      await dir.delete();
    }
  } on FileSystemException {
    // Gone already, or not ours to delete; the sweep at the next start tries again.
  }
}

/// What the pickers and the app name their copies: `salus-*`, image_picker's `image_picker_*` (iOS) and `scaled_*`,
/// file_picker's and share_plus's folders (Android), and the UUID-named files and folders both write.
final _copyName = RegExp(
  r'^(salus-|image_picker|scaled_|file_picker$|share_plus$|[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})',
  caseSensitive: false,
);

/// Deletes the copies an earlier run left in the app's own directories; everything else there stays.
Future<void> sweepCopies(Directory temp) async {
  for (final dir in _appDirectories(temp)) {
    final d = Directory(dir);
    if (!d.existsSync()) continue;
    try {
      for (final e in d.listSync()) {
        if (_copyName.hasMatch(e.path.split('/').last)) {
          await e.delete(recursive: true);
        }
      }
    } on FileSystemException {
      // Tried again at the next start.
    }
  }
}
