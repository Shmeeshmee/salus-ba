import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

import 'support.dart';

Future<Harness> signedOut(WidgetTester tester) async {
  final h = Harness();
  await h.pump(tester);
  return h;
}

Future<void> signIn(WidgetTester tester) async {
  await enter(tester, 'E-mail', email);
  await enter(tester, 'Password', password);
  await tapButton(tester, 'Sign in');
}

void main() {
  testWidgets('asks for the e-mail and password before calling the API', (tester) async {
    final h = await signedOut(tester);
    await tapButton(tester, 'Sign in');

    expect(find.text('Enter a valid e-mail address.'), findsOneWidget);
    expect(find.text('Enter your password.'), findsOneWidget);
    expect(h.api.requests, isEmpty);
  });

  testWidgets('a patient without MFA is signed in and the token is stored', (tester) async {
    final h = await signedOut(tester);
    h.api.answer('POST /auth/login', 200, loginDone());
    h.api.answer('GET /auth/me', 200, account());
    await signIn(tester);

    expect(find.text('Signed in as $email'), findsOneWidget);
    expect(h.store.token, token);
    // Nothing chosen on the device: no language, and the device's matched language (docs/i18n/SPEC.md §3).
    expect(jsonDecode(h.api.last('POST /auth/login').body), {
      'email': email,
      'password': password,
      'language': null,
      'detectedLanguage': 'en',
    });
    expect(h.api.last('GET /auth/me').headers['Authorization'], 'Bearer $token');
    expectNoCsrfHeader(h.api);
  });

  testWidgets('prints nothing, so no e-mail, code or token reaches a log', (tester) async {
    final printed = <String?>[];
    final original = debugPrint;
    debugPrint = (String? message, {int? wrapWidth}) => printed.add(message);
    try {
      final h = await signedOut(tester);
      h.api.answer('POST /auth/login', 200, loginNext('verify'));
      h.api.answer('POST /auth/mfa/verify', 200, loginDone());
      h.api.answer('GET /auth/me', 200, account());
      await signIn(tester);
      await enter(tester, 'Code', '123456');
      await tapButton(tester, 'Continue');

      expect(find.text('Signed in as $email'), findsOneWidget);
    } finally {
      debugPrint = original; // restored inside the test: the binding checks it before tear-down
    }
    expect(printed, isEmpty);
  });

  testWidgets('a wrong password says so and stores nothing', (tester) async {
    final h = await signedOut(tester);
    h.api.answer('POST /auth/login', 401, error('wrong_credentials'));
    await signIn(tester);

    expect(find.text('The e-mail or password is wrong.'), findsOneWidget);
    expect(h.store.token, isNull);
  });

  testWidgets('429 tells how long to wait from Retry-After', (tester) async {
    final h = await signedOut(tester);
    h.api.answer('POST /auth/login', 429, error('rate_limited'), {'retry-after': '540'});
    await signIn(tester);

    expect(find.text('Too many attempts. Try again in 9 minutes.'), findsOneWidget);
    expect(h.store.token, isNull);
  });

  testWidgets('429 without Retry-After says to try later', (tester) async {
    final h = await signedOut(tester);
    h.api.answer('POST /auth/login', 429, error('rate_limited'));
    await signIn(tester);

    expect(find.text('Too many attempts. Try again later.'), findsOneWidget);
  });

  testWidgets('an unreachable API says so', (tester) async {
    final h = await signedOut(tester);
    h.api.on('POST /auth/login', (_) async => throw http.ClientException('network'));
    await signIn(tester);

    expect(find.text('Salus could not be reached. Check your connection and try again.'), findsOneWidget);
  });

  group('a patient with MFA', () {
    Future<Harness> atCodeStep(WidgetTester tester) async {
      final h = await signedOut(tester);
      h.api.answer('POST /auth/login', 200, loginNext('verify'));
      await signIn(tester);
      return h;
    }

    testWidgets('is asked for the code and signed in with it', (tester) async {
      final h = await atCodeStep(tester);
      expect(find.text('Enter the 6-digit code from your authenticator app.'), findsOneWidget);
      expect(h.store.token, isNull);

      h.api.answer('POST /auth/mfa/verify', 200, loginDone());
      h.api.answer('GET /auth/me', 200, account());
      await enter(tester, 'Code', '123456');
      await tapButton(tester, 'Continue');

      expect(find.text('Signed in as $email'), findsOneWidget);
      expect(h.store.token, token);
      expect(h.api.calls, ['POST /auth/login', 'POST /auth/mfa/verify', 'GET /auth/me', 'GET /me/patient']);
      expect(jsonDecode(h.api.last('POST /auth/mfa/verify').body), {'challenge': challenge, 'code': '123456'});
      expectNoCsrfHeader(h.api);
    });

    testWidgets('a code that is not six digits is not sent', (tester) async {
      final h = await atCodeStep(tester);
      await enter(tester, 'Code', '12a45');
      await tapButton(tester, 'Continue');

      expect(find.text('The code has 6 digits.'), findsOneWidget);
      expect(h.api.calls, ['POST /auth/login']);
    });

    testWidgets('a wrong code can be tried again', (tester) async {
      final h = await atCodeStep(tester);
      h.api.answer('POST /auth/mfa/verify', 401, error('wrong_code'));
      await enter(tester, 'Code', '000000');
      await tapButton(tester, 'Continue');

      expect(find.text('The code is wrong. Check the time on your device and try again.'), findsOneWidget);
      expect(find.widgetWithText(FilledButton, 'Continue'), findsOneWidget);
      expect(h.store.token, isNull);
    });

    testWidgets('an ended challenge goes back to the password', (tester) async {
      final h = await atCodeStep(tester);
      h.api.answer('POST /auth/mfa/verify', 401, error('invalid_challenge'));
      await enter(tester, 'Code', '000000');
      await tapButton(tester, 'Continue');

      expect(find.text('The sign-in has ended. Enter your e-mail and password again.'), findsOneWidget);
      expect(find.widgetWithText(FilledButton, 'Sign in'), findsOneWidget);
      expect(h.store.token, isNull);
    });

    testWidgets('can go back to use another account', (tester) async {
      await atCodeStep(tester);
      await tapLink(tester, 'Use another account');

      expect(find.widgetWithText(FilledButton, 'Sign in'), findsOneWidget);
    });
  });

  group('clinic staff', () {
    const staffOnly = 'This app is for patients. Clinic staff use the Salus web app.';

    testWidgets('who must set up MFA are sent to the web app without enrolling', (tester) async {
      final h = await signedOut(tester);
      h.api.answer('POST /auth/login', 200, loginNext('enrol'));
      await signIn(tester);

      expect(find.text(staffOnly), findsOneWidget);
      expect(h.api.calls, ['POST /auth/login']);
      expect(h.store.token, isNull);
    });

    testWidgets('who verify a code are signed out again and nothing is stored', (tester) async {
      final h = await signedOut(tester);
      h.api.answer('POST /auth/login', 200, loginNext('verify'));
      h.api.answer('POST /auth/mfa/verify', 200, loginDone());
      h.api.answer('GET /auth/me', 200, account(role: 'admin'));
      h.api.answer('POST /auth/logout', 204);
      await signIn(tester);
      await enter(tester, 'Code', '123456');
      await tapButton(tester, 'Continue');

      expect(find.text(staffOnly), findsOneWidget);
      expect(h.api.last('POST /auth/logout').headers['Authorization'], 'Bearer $token');
      expect(h.store.token, isNull);
    });
  });
}
