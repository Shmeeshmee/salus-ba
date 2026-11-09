import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:salus/l10n/app_localizations.dart';
import 'package:salus/l10n/language.dart';

import 'support.dart';

// The language picker (docs/i18n/SPEC.md §3 "Picking and saving", COPY.md L1 to L4): before sign-in the choice is
// kept on the device and sent with the next sign-in or registration; signed in it is saved on the account at once,
// and the account's language wins over the device's.

AppLocalizations _in(String tag) => lookupAppLocalizations(localeOf(tag));

void _device(WidgetTester tester, List<Locale> locales) {
  tester.platformDispatcher.localesTestValue = locales;
  addTearDown(tester.platformDispatcher.clearLocalesTestValue);
}

/// Opens the picker from the screen's language button (its tooltip is L2 in [shown]) and picks [option].
Future<void> pick(WidgetTester tester, String option, {String shown = 'en'}) async {
  await tapFound(tester, find.byTooltip(_in(shown).language));
  await tapFound(tester, find.text(option));
}

Future<Harness> signedIn(WidgetTester tester, {String? language, FakeLanguageStore? languages}) async {
  final h = Harness(store: FakeTokenStore(token), languages: languages);
  h.api.answer('GET /auth/me', 200, {...account(), 'language': ?language});
  await h.pump(tester);
  return h;
}

void main() {
  testWidgets('the picker names every language in its own words, after the device language', (tester) async {
    await Harness().pump(tester);
    await tapFound(tester, find.byTooltip('Language'));

    expect(find.text('Language'), findsWidgets);
    expect(find.text("Use the device's language"), findsOneWidget);
    for (final name in ['Bosanski', 'Hrvatski', 'Srpski (latinica)', 'Српски (ћирилица)', 'English', 'Deutsch']) {
      expect(find.text(name), findsWidgets, reason: name);
    }
    for (final name in ['العربية', 'Türkçe', 'Italiano', 'Français']) {
      expect(find.text(name), findsOneWidget, reason: name);
    }
  });

  testWidgets('before sign-in: the choice switches the app at once and is kept on the device', (tester) async {
    final h = Harness();
    await h.pump(tester);
    await pick(tester, 'Hrvatski');

    expect(find.text(_in('hr').signInTitle), findsOneWidget);
    expect(h.languages.language, 'hr');
    expect(h.api.requests, isEmpty);
  });

  testWidgets('the choice and the matched device language go with the sign-in', (tester) async {
    _device(tester, const [Locale('de', 'AT')]);
    final h = Harness();
    await h.pump(tester);
    await pick(tester, 'Hrvatski', shown: 'de');
    h.api.answer('POST /auth/login', 200, loginDone());
    h.api.answer('GET /auth/me', 200, {...account(), 'language': 'hr'});
    await enter(tester, _in('hr').email, email);
    await enter(tester, _in('hr').password, password);
    await tapButton(tester, _in('hr').signIn);

    expect(jsonDecode(h.api.last('POST /auth/login').body), {
      'email': email,
      'password': password,
      'language': 'hr',
      'detectedLanguage': 'de',
    });
  });

  testWidgets('the choice and the matched device language go with the registration', (tester) async {
    _device(tester, const [Locale('ar', 'SA')]);
    final h = Harness();
    await h.pump(tester);
    await tapLink(tester, _in('ar').createAnAccount);
    await pick(tester, 'Türkçe', shown: 'ar');
    h.api.answer('POST /auth/register', 201, account());
    h.api.answer('POST /auth/login', 200, loginDone());
    h.api.answer('GET /auth/me', 200, {...account(), 'language': 'tr'});
    await enter(tester, _in('tr').email, email);
    await enter(tester, _in('tr').password, password);
    await tapButton(tester, _in('tr').createAccount);

    final sent = {'email': email, 'password': password, 'language': 'tr', 'detectedLanguage': 'ar'};
    expect(jsonDecode(h.api.last('POST /auth/register').body), sent);
    expect(jsonDecode(h.api.last('POST /auth/login').body), sent);
  });

  testWidgets("the device's language clears the choice on the device", (tester) async {
    _device(tester, const [Locale('it')]);
    final h = Harness(languages: FakeLanguageStore('hr'));
    await h.pump(tester);
    expect(find.text(_in('hr').signInTitle), findsOneWidget);

    await pick(tester, _in('hr').useDeviceLanguage, shown: 'hr');

    expect(find.text(_in('it').signInTitle), findsOneWidget);
    expect(h.languages.language, isNull);
  });

  testWidgets("signed in: the choice is saved on the account and the app says so", (tester) async {
    final h = await signedIn(tester);
    h.api.answer('PATCH /auth/me', 200, {...account(), 'language': 'de'});
    await pick(tester, 'Deutsch');

    expect(jsonDecode(h.api.last('PATCH /auth/me').body), {'language': 'de'});
    expect(find.text(_in('de').languageSaved), findsOneWidget);
    expect(find.text(_in('de').signOut), findsOneWidget);
    expect(h.languages.language, 'de');
  });

  testWidgets("signed in: the device's language clears the account's choice", (tester) async {
    _device(tester, const [Locale('fr')]);
    final h = await signedIn(tester, language: 'de', languages: FakeLanguageStore('de'));
    h.api.answer('PATCH /auth/me', 200, account());
    await pick(tester, _in('de').useDeviceLanguage, shown: 'de');

    expect(jsonDecode(h.api.last('PATCH /auth/me').body), {'language': null});
    expect(find.text(_in('fr').languageSaved), findsOneWidget);
    expect(find.text(_in('fr').signOut), findsOneWidget);
    expect(h.languages.language, isNull);
  });

  testWidgets('signed in: a save that fails says so and keeps the choice on this device', (tester) async {
    final h = await signedIn(tester);
    h.api.on('PATCH /auth/me', (_) async => throw http.ClientException('network'));
    await pick(tester, 'Italiano');

    expect(find.text(_in('it').unreachable), findsOneWidget);
    expect(find.text(_in('it').signOut), findsOneWidget);
    expect(h.languages.language, 'it');
  });

  testWidgets("the account's language wins over the device's once signed in, and is kept on the device", (
    tester,
  ) async {
    _device(tester, const [Locale('fr')]);
    final h = await signedIn(tester, language: 'sr-Cyrl', languages: FakeLanguageStore('de'));

    expect(find.text(_in('sr-Cyrl').signOut), findsOneWidget);
    expect(h.languages.language, 'sr-Cyrl');
  });

  testWidgets("an account without a language follows the device's on every device", (tester) async {
    _device(tester, const [Locale('fr')]);
    final h = await signedIn(tester, languages: FakeLanguageStore('de'));

    expect(find.text(_in('fr').signOut), findsOneWidget);
    expect(h.languages.language, isNull);
  });

  testWidgets('the account chosen at sign-in sets the language', (tester) async {
    final h = Harness();
    await h.pump(tester);
    h.api.answer('POST /auth/login', 200, loginDone());
    h.api.answer('GET /auth/me', 200, {...account(), 'language': 'it'});
    await enter(tester, 'E-mail', email);
    await enter(tester, 'Password', password);
    await tapButton(tester, 'Sign in');

    expect(find.text(_in('it').signOut), findsOneWidget);
    expect(h.languages.language, 'it');
  });
}
