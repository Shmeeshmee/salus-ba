import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:intl/intl.dart' show DateFormat, NumberFormat;
import 'package:salus/l10n/language.dart';

import 'support.dart';

void main() {
  group('the device language is matched to an app language (SPEC §3 "Matching")', () {
    final cases = <(String, List<Locale>, String)>[
      ('Bosnian', [const Locale('bs', 'BA')], 'bs'),
      ('Croatian', [const Locale('hr')], 'hr'),
      (
        'Serbian in Latin script',
        [const Locale.fromSubtags(languageCode: 'sr', scriptCode: 'Latn', countryCode: 'RS')],
        'sr-Latn',
      ),
      ('Serbian in Montenegro', [const Locale('sr', 'ME')], 'sr-Latn'),
      ('Serbo-Croatian', [const Locale('sh')], 'sr-Latn'),
      ('Montenegrin', [const Locale('cnr')], 'sr-Latn'),
      ('Serbian without a script', [const Locale('sr')], 'sr-Cyrl'),
      ('Serbian in Serbia', [const Locale('sr', 'RS')], 'sr-Cyrl'),
      ('Serbian in Bosnia', [const Locale('sr', 'BA')], 'sr-Cyrl'),
      (
        'Serbian in Cyrillic script',
        [const Locale.fromSubtags(languageCode: 'sr', scriptCode: 'Cyrl', countryCode: 'ME')],
        'sr-Cyrl',
      ),
      ('English', [const Locale('en', 'GB')], 'en'),
      ('German', [const Locale('de', 'AT')], 'de'),
      ('Arabic', [const Locale('ar', 'SA')], 'ar'),
      ('Turkish', [const Locale('tr')], 'tr'),
      ('Italian', [const Locale('it', 'CH')], 'it'),
      ('French', [const Locale('fr', 'BE')], 'fr'),
      ('the first match in the device order', [const Locale('ja'), const Locale('de'), const Locale('hr')], 'de'),
      ('nothing matches', [const Locale('ja'), const Locale('es')], 'en'),
      ('no device language', const [], 'en'),
    ];
    for (final (name, device, tag) in cases) {
      test(name, () => expect(matchLanguage(device), tag));
    }
  });

  test('every language has a locale the app supports, Serbian by script', () {
    expect(languages, ['bs', 'hr', 'sr-Latn', 'sr-Cyrl', 'en', 'de', 'ar', 'tr', 'it', 'fr']);
    expect(localeOf('sr-Latn'), const Locale.fromSubtags(languageCode: 'sr', scriptCode: 'Latn'));
    expect(localeOf('sr-Cyrl'), const Locale.fromSubtags(languageCode: 'sr', scriptCode: 'Cyrl'));
    expect(localeOf('ar'), const Locale('ar'));
  });

  group('the app speaks the language', () {
    Future<Harness> failedStart(WidgetTester tester, {List<Locale>? device, String? saved}) async {
      if (device != null) {
        tester.platformDispatcher.localesTestValue = device;
        addTearDown(tester.platformDispatcher.clearLocalesTestValue);
      }
      final h = Harness(store: FakeTokenStore(token), languages: FakeLanguageStore(saved));
      h.api.on('GET /auth/me', (_) async => throw http.ClientException('network'));
      await h.pump(tester);
      return h;
    }

    final tryAgain = {
      'bs': 'Pokušajte ponovo',
      'hr': 'Pokušajte ponovno',
      'sr-Latn': 'Pokušajte ponovo',
      'sr-Cyrl': 'Покушајте поново',
      'en': 'Try again',
      'de': 'Erneut versuchen',
      'ar': 'إعادة المحاولة',
      'tr': 'Tekrar dene',
      'it': 'Riprova',
      'fr': 'Réessayer',
    };
    for (final MapEntry(key: tag, value: text) in tryAgain.entries) {
      testWidgets('of the device: $tag', (tester) async {
        await failedStart(tester, device: [localeOf(tag)]);
        expect(find.widgetWithText(FilledButton, text), findsOneWidget);
      });
    }

    testWidgets('English on a device whose language the app does not have', (tester) async {
      await failedStart(tester, device: const [Locale('ja')]);
      expect(find.widgetWithText(FilledButton, 'Try again'), findsOneWidget);
    });

    testWidgets('the language saved on the device wins over the device language', (tester) async {
      await failedStart(tester, device: const [Locale('de')], saved: 'hr');
      expect(find.widgetWithText(FilledButton, 'Pokušajte ponovno'), findsOneWidget);
    });

    testWidgets('a saved value that is not an app language is ignored', (tester) async {
      await failedStart(tester, device: const [Locale('de')], saved: 'xx');
      expect(find.widgetWithText(FilledButton, 'Erneut versuchen'), findsOneWidget);
    });

    testWidgets('storage that cannot be read leaves the device language', (tester) async {
      tester.platformDispatcher.localesTestValue = const [Locale('it')];
      addTearDown(tester.platformDispatcher.clearLocalesTestValue);
      final h = Harness(store: FakeTokenStore(token), languages: FakeLanguageStore()..fails = true);
      h.api.on('GET /auth/me', (_) async => throw http.ClientException('network'));
      await h.pump(tester);
      expect(find.widgetWithText(FilledButton, 'Riprova'), findsOneWidget);
    });

    testWidgets('Arabic is right to left and other languages left to right', (tester) async {
      await failedStart(tester, device: const [Locale('ar')]);
      expect(Directionality.of(tester.element(find.byType(FilledButton))), TextDirection.rtl);

      await failedStart(tester, device: const [Locale('sr', 'BA')]);
      expect(Directionality.of(tester.element(find.byType(FilledButton))), TextDirection.ltr);
    });

    testWidgets('Arabic times and numbers have Latin digits and a 24-hour clock', (tester) async {
      await failedStart(tester, device: const [Locale('ar')]);
      final locale = Localizations.localeOf(tester.element(find.byType(FilledButton))).toString();
      expect(DateFormat.Hm(locale).format(DateTime(2026, 11, 4, 9, 5)), '09:05');
      expect(DateFormat.Hm(locale).format(DateTime(2026, 11, 4, 21, 30)), '21:30');
      expect(NumberFormat.decimalPattern(locale).format(1234567), contains('1'));
      expect(NumberFormat.decimalPattern(locale).format(1234567), isNot(contains('١')));
    });

    testWidgets('Arabic letters fall back to the bundled Noto Sans Arabic', (tester) async {
      await failedStart(tester, device: const [Locale('ar')]);
      final theme = Theme.of(tester.element(find.byType(FilledButton)));
      expect(theme.textTheme.bodyMedium!.fontFamilyFallback, contains(arabicFont));
      expect(theme.textTheme.labelLarge!.fontFamilyFallback, contains(arabicFont));
      expect(theme.textTheme.headlineSmall!.fontFamilyFallback, contains(arabicFont));
    });
  });

  test('the Arabic font is bundled from the app, with its licence', () {
    final pubspec = File('pubspec.yaml').readAsStringSync();
    expect(pubspec, contains('family: $arabicFont'));
    for (final weight in ['Regular', 'Medium', 'Bold']) {
      expect(File('assets/fonts/NotoSansArabic-$weight.ttf').existsSync(), isTrue, reason: weight);
      expect(pubspec, contains('asset: assets/fonts/NotoSansArabic-$weight.ttf'));
    }
    expect(File('assets/fonts/NotoSansArabic-OFL.txt').readAsStringSync(), contains('SIL Open Font License'));
  });

  test('keeps the chosen language in secure storage, apart from the session token, and clears it', () async {
    TestWidgetsFlutterBinding.ensureInitialized();
    FlutterSecureStorage.setMockInitialValues({'session_token': token});
    const store = SecureLanguageStore(FlutterSecureStorage());

    expect(await store.read(), isNull);
    await store.write('sr-Cyrl');
    expect(await store.read(), 'sr-Cyrl');
    expect(await const FlutterSecureStorage().readAll(), {'session_token': token, 'language': 'sr-Cyrl'});

    await store.delete();
    expect(await store.read(), isNull);
    expect(await const FlutterSecureStorage().readAll(), {'session_token': token});
  });

  test('the Serbian fallback catalog gen-l10n requires is the Cyrillic one, word for word', () {
    Map<String, dynamic> read(String name) =>
        jsonDecode(File('lib/l10n/$name').readAsStringSync()) as Map<String, dynamic>;
    final fallback = read('app_sr.arb')..remove('@@locale');
    final cyrillic = read('app_sr_Cyrl.arb')..remove('@@locale');
    expect(fallback, cyrillic);
  });
}
