import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';

import 'support.dart';

Future<Harness> atRegister(WidgetTester tester) async {
  final h = Harness();
  await h.pump(tester);
  await tapLink(tester, 'Create an account');
  return h;
}

Future<void> register(WidgetTester tester, {String mail = email, String pass = password}) async {
  await enter(tester, 'E-mail', mail);
  await enter(tester, 'Password', pass);
  await tapButton(tester, 'Create account');
}

void main() {
  testWidgets('checks the e-mail and the password length before calling the API', (tester) async {
    final cases = {
      'no e-mail': ('', password, 'Enter a valid e-mail address.'),
      'not an e-mail': ('patient.example.test', password, 'Enter a valid e-mail address.'),
      '11 characters': (email, 'a' * 11, 'Use 12 to 128 characters for the password.'),
      '129 characters': (email, 'a' * 129, 'Use 12 to 128 characters for the password.'),
    };
    for (final MapEntry(key: name, value: (mail, pass, message)) in cases.entries) {
      final h = await atRegister(tester);
      await register(tester, mail: mail, pass: pass);

      expect(find.text(message), findsOneWidget, reason: name);
      expect(h.api.requests, isEmpty, reason: name);
    }
  });

  testWidgets('counts the password in characters, not bytes', (tester) async {
    final h = await atRegister(tester);
    h.api.answer('POST /auth/register', 201, account());
    h.api.answer('POST /auth/login', 200, loginDone());
    h.api.answer('GET /auth/me', 200, account());
    await register(tester, pass: 'ü' * 12); // 24 bytes in UTF-8, 12 characters

    expect(h.api.calls.first, 'POST /auth/register');
  });

  testWidgets('registers, signs in and stores the token', (tester) async {
    final h = await atRegister(tester);
    h.api.answer('POST /auth/register', 201, account());
    h.api.answer('POST /auth/login', 200, loginDone());
    h.api.answer('GET /auth/me', 200, account());
    await register(tester);

    expect(find.text('Signed in as $email'), findsOneWidget);
    expect(h.store.token, token);
    expect(h.api.calls, ['POST /auth/register', 'POST /auth/login', 'GET /auth/me']);
    expect(jsonDecode(h.api.last('POST /auth/register').body), {'email': email, 'password': password});
    expectNoCsrfHeader(h.api);
  });

  testWidgets('409 says the e-mail has an account', (tester) async {
    final h = await atRegister(tester);
    h.api.answer('POST /auth/register', 409, error('email_taken'));
    await register(tester);

    expect(find.text('This e-mail already has an account. Sign in instead.'), findsOneWidget);
    expect(h.api.calls, ['POST /auth/register']);
    expect(h.store.token, isNull);
  });

  testWidgets('the API refusing the e-mail or password is shown as a fixed message', (tester) async {
    final cases = {
      'invalid_email': 'Enter a valid e-mail address.',
      'invalid_password': 'Use 12 to 128 characters for the password.',
      'invalid_request': 'Something went wrong. Try again.',
    };
    for (final MapEntry(key: code, value: message) in cases.entries) {
      final h = await atRegister(tester);
      h.api.answer('POST /auth/register', 400, error(code));
      await register(tester);

      expect(find.text(message), findsOneWidget, reason: code);
    }
  });

  testWidgets('when signing in after registering fails it goes to sign-in with a notice', (tester) async {
    final h = await atRegister(tester);
    h.api.answer('POST /auth/register', 201, account());
    h.api.answer('POST /auth/login', 429, error('rate_limited'), {'retry-after': '60'});
    await register(tester);

    expect(find.text('Your account is ready. Sign in to continue.'), findsOneWidget);
    expect(find.text('Sign in to Salus'), findsOneWidget);
    expect(h.store.token, isNull);
  });

  testWidgets('can go back to sign in', (tester) async {
    await atRegister(tester);
    await tapLink(tester, 'I already have an account');

    expect(find.text('Sign in to Salus'), findsOneWidget);
  });
}
