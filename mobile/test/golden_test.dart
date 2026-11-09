import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:salus/call.dart';
import 'package:salus/l10n/app_localizations.dart';
import 'package:salus/l10n/language.dart';

import 'book_test.dart' show slotsByDay;
import 'call_test.dart' show joinRoute, joined, status, statusRoute;
import 'support.dart';

// Screens in Arabic (right to left) and Serbian Cyrillic, as golden screenshots (docs/i18n/IMPLEMENTATION_PLAN.md
// 1.7): sign-in, My appointments, booking, the consent screen and the call. Widget tests draw text in a test font
// of boxes, so the real fonts are loaded first: Roboto and the Material icons from the Flutter SDK, and the
// bundled Noto Sans Arabic. `flutter test --update-goldens test/golden_test.dart` writes them again; review the
// images before committing.

AppLocalizations _in(String tag) => lookupAppLocalizations(localeOf(tag));

/// The Flutter SDK's own fonts (bin/cache/artifacts/material_fonts), found from the test runner's path.
Directory _sdkFonts() {
  final root = Platform.environment['FLUTTER_ROOT'];
  if (root != null) return Directory('$root/bin/cache/artifacts/material_fonts');
  var dir = File(Platform.resolvedExecutable).parent;
  while (!Directory('${dir.path}/material_fonts').existsSync()) {
    dir = dir.parent;
  }
  return Directory('${dir.path}/material_fonts');
}

Future<void> _load(String family, List<String> paths) async {
  final loader = FontLoader(family);
  for (final path in paths) {
    loader.addFont(Future.value(ByteData.sublistView(File(path).readAsBytesSync())));
  }
  await loader.load();
}

Future<void> _loadFonts() async {
  final sdk = _sdkFonts().path;
  await _load('Roboto', ['$sdk/Roboto-Regular.ttf', '$sdk/Roboto-Medium.ttf', '$sdk/Roboto-Bold.ttf']);
  await _load('MaterialIcons', ['$sdk/MaterialIcons-Regular.otf']);
  await _load(arabicFont, [
    'assets/fonts/NotoSansArabic-Regular.ttf',
    'assets/fonts/NotoSansArabic-Medium.ttf',
    'assets/fonts/NotoSansArabic-Bold.ttf',
  ]);
}

/// A 390 × 844 phone, at one pixel per point so the images stay small.
void _phone(WidgetTester tester) {
  tester.view.physicalSize = const Size(390, 844);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
}

Harness _signedIn(String tag) {
  final h = Harness(store: FakeTokenStore(token), languages: FakeLanguageStore(tag));
  h.api.answer('GET /auth/me', 200, {...account(), 'language': tag});
  answerDirectory(h.api);
  h.api.on('GET /fhir/R4/Schedule', (_) async => fhirResponse(200, bundle([schedule()])));
  h.api.on('GET /fhir/R4/Slot', slotsByDay);
  h.api.answer('GET /me/consents', 200, {
    'consents': [consent('c1')],
  });
  h.api.on(
    'GET /fhir/R4/Appointment',
    (_) async => fhirResponse(
      200,
      bundle([
        appointment('now', '2026-11-04T09:05:00Z', '2026-11-04T09:35:00Z'),
        appointment('next', '2026-11-12T13:00:00Z', '2026-11-12T13:30:00Z'),
        appointment('past', '2026-11-02T08:00:00Z', '2026-11-02T08:30:00Z', status: 'fulfilled'),
      ]),
    ),
  );
  h.api.answer(joinRoute, 200, joined());
  h.api.answer(statusRoute, 200, status('open'));
  h.api.answer('POST /appointments', 409, error('consent_required'));
  return h;
}

Future<void> _golden(WidgetTester tester, String name) =>
    expectLater(find.byType(MaterialApp), matchesGoldenFile('goldens/$name.png'));

void main() {
  setUpAll(_loadFonts);

  for (final tag in ['ar', 'sr-Cyrl']) {
    group(tag, () {
      final t = _in(tag);

      testWidgets('sign-in', (tester) async {
        _phone(tester);
        await Harness(languages: FakeLanguageStore(tag)).pump(tester);
        await _golden(tester, 'sign_in_$tag');
      });

      testWidgets('my appointments', (tester) async {
        _phone(tester);
        await _signedIn(tag).pump(tester);
        await tapOutlined(tester, t.myAppointments);
        await _golden(tester, 'appointments_$tag');
      });

      testWidgets('booking', (tester) async {
        _phone(tester);
        await _signedIn(tag).pump(tester);
        await tapButton(tester, t.bookAppointment);
        await tapText(tester, clinicName);
        await tapText(tester, 'Cy Example');
        await _golden(tester, 'booking_$tag');
      });

      testWidgets('consent', (tester) async {
        _phone(tester);
        await _signedIn(tag).pump(tester);
        await tapButton(tester, t.bookAppointment);
        await tapText(tester, clinicName);
        await tapText(tester, 'Cy Example');
        await tapOutlined(tester, '14:00');
        await tapButton(tester, t.book);
        await _golden(tester, 'consent_$tag');
      });

      testWidgets('call', (tester) async {
        _phone(tester);
        final h = _signedIn(tag);
        await h.pump(tester);
        await tapOutlined(tester, t.myAppointments);
        await tapButton(tester, t.joinConsultation);
        await tapButton(tester, t.joinNow);
        h.media.room.emit(const PeerChanged(Peer(present: true, camera: true, microphone: true)));
        await tester.pumpAndSettle();
        await _golden(tester, 'call_$tag');
      });
    });
  }

  testWidgets('a back arrow points the way the language reads', (tester) async {
    await _signedIn('ar').pump(tester);
    await tapOutlined(tester, _in('ar').myAppointments);
    final back = tester.widget<Icon>(find.descendant(of: find.byType(BackButton), matching: find.byType(Icon)));
    expect(Directionality.of(tester.element(find.byType(BackButton))), TextDirection.rtl);
    expect(back.icon!.matchTextDirection, isTrue);
  });

  testWidgets('phone numbers, codes and e-mail addresses are typed left to right in Arabic', (tester) async {
    await Harness(languages: FakeLanguageStore('ar')).pump(tester);
    final email = tester.widget<TextField>(
      find.descendant(of: find.widgetWithText(TextFormField, _in('ar').email), matching: find.byType(TextField)),
    );
    expect(email.textDirection, TextDirection.ltr);
  });

  testWidgets("specialties are named from the catalog by their SNOMED code, not the API's English", (tester) async {
    final h = _signedIn('de');
    await h.pump(tester);
    await tapButton(tester, _in('de').bookAppointment);
    await tapText(tester, clinicName);

    expect(find.text(_in('de').specialty394579002), findsOneWidget);
    expect(find.text('Cardiology'), findsNothing);
  });
}
