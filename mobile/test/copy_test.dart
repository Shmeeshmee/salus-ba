import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

// The mobile text of the video consultation is the owner's, word for word (docs/consultation/COPY.md): every M
// string is in the English catalog (lib/l10n/app_en.arb, the template of every language), and the iOS permission
// text is in Info.plist.

/// The English catalog's texts, each plural's branches as texts of their own, placeholders as `{name}`.
final _english = () {
  final arb = jsonDecode(File('lib/l10n/app_en.arb').readAsStringSync()) as Map<String, dynamic>;
  final branch = RegExp(r'(?:zero|one|two|few|many|other|=\d+)\{((?:[^{}]|\{\w+\})*)\}');
  return {
    for (final MapEntry(:key, :value) in arb.entries)
      if (!key.startsWith('@')) ...[value as String, for (final m in branch.allMatches(value)) m[1]!],
  };
}();

/// Whether the catalog has [s] word for word, each of its `[placeholder]`s as any `{name}`.
bool _inCatalog(String s) {
  final pattern = RegExp('^${RegExp.escape(s).replaceAll(RegExp(r'\\\[[^\]]+\\\]'), r'\{\w+\}')}\$');
  return _english.any(pattern.hasMatch);
}

final _copy = File('../docs/consultation/COPY.md').readAsLinesSync();

/// docs/attachments/COPY.md one bullet per line: its strings run on over the indented lines after the dash.
final _attachments = [
  for (final line in File('../docs/attachments/COPY.md').readAsLinesSync().join('\n').split(RegExp(r'\n(?!  )')))
    line.replaceAll(RegExp(r'\n  +'), ' '),
];

/// The quoted strings of the lines that name an M string ("- M1: …", "- W1 / M1: …"), placeholders as `[name]`.
List<String> _mobileStrings() => [
  for (final line in _copy)
    if (RegExp(r'^- (W\d+[a-z]? / )?M\d+[ :]').hasMatch(line) || line.contains('(M35)'))
      for (final m in RegExp(r'"([^"]+)"').allMatches(line))
        if (!RegExp(r'^[A-Z]\d+').hasMatch(m[1]!)) m[1]!,
];

void main() {
  test('every mobile string of COPY.md is in the English catalog', () {
    final strings = _mobileStrings();

    expect(strings, hasLength(greaterThan(35)), reason: 'COPY.md parsed');
    for (final s in strings) {
      expect(_inCatalog(s), isTrue, reason: s);
    }
  });

  test('every mobile string of the questionnaire COPY.md is in the English catalog, placeholders as parameters', () {
    final strings = [
      for (final line in File('../docs/questionnaire/COPY.md').readAsLinesSync())
        if (RegExp(r'^- M\d+[ :]').hasMatch(line))
          for (final m in RegExp(r'"([^"]+)"').allMatches(line))
            // "[Questionnaire title]" alone is the title itself, not a sentence.
            if (!RegExp(r'^\[[^\]]+\]$').hasMatch(m[1]!)) m[1]!,
    ];

    expect(strings, hasLength(greaterThan(45)), reason: 'COPY.md parsed');
    for (final s in strings) {
      expect(_inCatalog(s), isTrue, reason: s);
    }
  });

  test('every mobile string of the attachments COPY.md is in the English catalog, placeholders as parameters', () {
    final strings = [
      for (final line in _attachments)
        // M15 to M17 are iOS purpose strings, in Info.plist.
        if (RegExp(r'^- M(\d+)[ :]').firstMatch(line) case final m? when !{15, 16, 17}.contains(int.parse(m[1]!)))
          for (final q in RegExp(r'"([^"]+)"').allMatches(line))
            // "[Title]" alone is the file's name itself, not a sentence.
            if (!RegExp(r'^\[[^\]]+\]$').hasMatch(q[1]!)) q[1]!,
    ];

    expect(strings, hasLength(greaterThan(45)), reason: 'COPY.md parsed');
    for (final s in strings) {
      expect(_inCatalog(s), isTrue, reason: s);
    }
  });

  test("every mobile string of the languages COPY.md is in the English catalog, placeholders as parameters", () {
    final strings = [
      for (final line in File('../docs/i18n/COPY.md').readAsLinesSync())
        if (RegExp(r'^- M\d+[ :]').hasMatch(line))
          for (final m in RegExp(r'"([^"]+)"').allMatches(line)) m[1]!,
    ];

    expect(strings, hasLength(4), reason: 'COPY.md parsed');
    for (final s in strings) {
      expect(_inCatalog(s), isTrue, reason: s);
    }
  });

  // docs/attachments/COPY.md M15 and M16 replace the consultation's two purpose strings (one per permission on
  // iOS), and M17 is new (owner, 2026-11-07).
  test("iOS asks for the camera, the microphone and the photo library in the owner's words", () {
    final plist = File('ios/Runner/Info.plist').readAsStringSync();
    for (final key in ['NSCameraUsageDescription', 'NSMicrophoneUsageDescription', 'NSPhotoLibraryUsageDescription']) {
      final line = _attachments.firstWhere((l) => l.contains('`$key`'));
      final text = RegExp(r'"([^"]+)"').firstMatch(line)![1]!;
      expect(plist, contains('<key>$key</key>\n\t<string>$text</string>'), reason: key);
    }
  });

  test('Android declares the camera and the microphone, and no camera is required', () {
    final manifest = File('android/app/src/main/AndroidManifest.xml').readAsStringSync();
    for (final p in ['CAMERA', 'RECORD_AUDIO', 'MODIFY_AUDIO_SETTINGS']) {
      expect(manifest, contains('<uses-permission android:name="android.permission.$p"/>'), reason: p);
    }
    expect(manifest, contains('<uses-feature android:name="android.hardware.camera" android:required="false"/>'));
  });
}
