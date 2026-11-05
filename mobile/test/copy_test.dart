import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

// The mobile text of the video consultation is the owner's, word for word (docs/consultation/COPY.md): every M
// string is in lib/messages.dart, and the iOS permission text is in Info.plist.
final _copy = File('../docs/consultation/COPY.md').readAsLinesSync();

/// The quoted strings of the lines that name an M string ("- M1: …", "- W1 / M1: …"), placeholders as `$name`.
List<String> _mobileStrings() => [
  for (final line in _copy)
    if (RegExp(r'^- (W\d+[a-z]? / )?M\d+[ :]').hasMatch(line) || line.contains('(M35)'))
      for (final m in RegExp(r'"([^"]+)"').allMatches(line))
        if (!RegExp(r'^[A-Z]\d+').hasMatch(m[1]!)) m[1]!.replaceAll(RegExp(r'\[[^\]]+\]'), r'$name'),
];

void main() {
  test('every mobile string of COPY.md is in messages.dart', () {
    final messages = File('lib/messages.dart').readAsStringSync().replaceAll("'\n    '", '');
    final strings = _mobileStrings();

    expect(strings, hasLength(greaterThan(35)), reason: 'COPY.md parsed');
    for (final s in strings) {
      expect(messages, anyOf(contains("'${s.replaceAll("'", r"\'")}'"), contains('"$s"')), reason: s);
    }
  });

  test("iOS asks for the camera and the microphone in the owner's words", () {
    final plist = File('ios/Runner/Info.plist').readAsStringSync();
    for (final key in ['NSCameraUsageDescription', 'NSMicrophoneUsageDescription']) {
      final line = _copy.firstWhere((l) => l.contains('`$key`'));
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
