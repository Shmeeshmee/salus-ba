import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:salus/language_picker.dart';

// Every user-facing string comes from the catalogs (docs/i18n/IMPLEMENTATION_PLAN.md, "No hard-coded user-facing
// string"); the iOS permission texts are localised per language (InfoPlist.strings).

/// The places a widget takes text, written with a string literal that has a letter in it.
final _literal = RegExp(
  r'''(\b(Text|Heading|Message|SubmitButton|Choice|SnackBar)\(\s*|\b(labelText|helperText|hintText|tooltip|title|label|semanticsLabel|helpText|cancelText|confirmText|errorFormatText|errorInvalidText|message|notice):\s*)(const\s+)?['"][^'"]*\p{L}''',
  unicode: true,
);

/// The brand name, the same in every language.
const _allowed = ["title: 'Salus'"];

/// The iOS folders of the ten languages: Serbian Cyrillic is iOS's plain `sr`.
const _lproj = ['bs', 'hr', 'sr-Latn', 'sr', 'en', 'de', 'ar', 'tr', 'it', 'fr'];

void main() {
  test('no widget is given a string literal: the text comes from AppLocalizations', () {
    final found = <String>[];
    for (final file in Directory('lib').listSync(recursive: true).whereType<File>()) {
      if (!file.path.endsWith('.dart') || file.path.startsWith('lib/l10n/')) continue;
      final lines = file.readAsLinesSync();
      for (var i = 0; i < lines.length; i++) {
        final line = lines[i];
        if (line.trimLeft().startsWith('//')) continue;
        if (_literal.hasMatch(line) && !_allowed.any(line.contains)) found.add('${file.path}:${i + 1}: ${line.trim()}');
      }
    }
    expect(found, isEmpty);
  });

  test('messages.dart maps codes to catalog lookups and writes no sentence of its own', () {
    final source = File('lib/messages.dart').readAsStringSync();
    expect(RegExp(r'''(=|=>)\s*['"][^'"]*\p{L}''', unicode: true).hasMatch(source), isFalse);
  });

  test('iOS offers the app in the ten languages', () {
    final plist = File('ios/Runner/Info.plist').readAsStringSync();
    final list = RegExp(r'<key>CFBundleLocalizations</key>\s*<array>(.*?)</array>', dotAll: true).firstMatch(plist);
    expect(list, isNotNull);
    expect(RegExp(r'<string>([^<]+)</string>').allMatches(list![1]!).map((m) => m[1]).toSet(), _lproj.toSet());
  });

  test("iOS asks for the camera, the microphone and the photos in every language", () {
    final project = File('ios/Runner.xcodeproj/project.pbxproj').readAsStringSync();
    for (final lang in _lproj) {
      final file = File('ios/Runner/$lang.lproj/InfoPlist.strings');
      expect(file.existsSync(), isTrue, reason: lang);
      final text = file.readAsStringSync();
      for (final key in [
        'NSCameraUsageDescription',
        'NSMicrophoneUsageDescription',
        'NSPhotoLibraryUsageDescription',
      ]) {
        expect(text, matches(RegExp('^"$key" = "[^"]+";\$', multiLine: true)), reason: '$lang $key');
      }
      // Bundled only when the Xcode project lists it.
      // Xcode quotes a value with a dash.
      final q = lang.contains('-') ? '"' : '';
      expect(
        project,
        contains(
          '/* $lang */ = {isa = PBXFileReference; lastKnownFileType = text.plist.strings; '
          'name = $q$lang$q; path = $q$lang.lproj/InfoPlist.strings$q; sourceTree = "<group>"; };',
        ),
        reason: lang,
      );
    }
    expect(project, contains('/* InfoPlist.strings in Resources */'));
  });

  test('the English InfoPlist.strings has the words of Info.plist', () {
    final plist = File('ios/Runner/Info.plist').readAsStringSync();
    final english = File('ios/Runner/en.lproj/InfoPlist.strings').readAsStringSync();
    for (final key in ['NSCameraUsageDescription', 'NSMicrophoneUsageDescription', 'NSPhotoLibraryUsageDescription']) {
      final text = RegExp('<key>$key</key>\\s*<string>([^<]+)</string>').firstMatch(plist)![1]!;
      expect(english, contains('"$key" = "$text";'), reason: key);
    }
  });

  test('the strings with legal weight carry the legal tag', () {
    final en = jsonDecode(File('lib/l10n/app_en.arb').readAsStringSync()) as Map<String, dynamic>;
    bool legal(String key) => ((en['@$key'] as Map?)?['description'] as String? ?? '').startsWith('[legal]');
    for (final key in [
      'consentIntro',
      'whatIsSharedBody',
      'whoSeesItBody',
      'whatForBody',
      'whoIsResponsibleBody',
      'howToRevokeBody',
      'readPrivacyNotice',
      'agreeAndBook',
      'revokeBody',
      'consentRevoked',
      'cancelBody',
      'cancelWindowHint',
      'whyCameraAndMicrophone',
      'whyCameraForFiles',
    ]) {
      expect(legal(key), isTrue, reason: key);
    }
  });

  test('every language name in the picker is written in its own language (COPY.md L1)', () {
    expect(languageNames, {
      'bs': 'Bosanski',
      'hr': 'Hrvatski',
      'sr-Latn': 'Srpski (latinica)',
      'sr-Cyrl': 'Српски (ћирилица)',
      'en': 'English',
      'de': 'Deutsch',
      'ar': 'العربية',
      'tr': 'Türkçe',
      'it': 'Italiano',
      'fr': 'Français',
    });
  });
}
