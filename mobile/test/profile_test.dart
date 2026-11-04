import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'support.dart';

Future<Harness> withoutProfile(WidgetTester tester) async {
  final h = Harness(store: FakeTokenStore(token));
  h.api.answer('GET /auth/me', 200, account());
  h.api.answer('GET /me/patient', 404, error('not_found'));
  await h.pump(tester);
  return h;
}

Future<void> fill(
  WidgetTester tester, {
  String given = 'Ada',
  String family = 'Example',
  String birth = '1990-02-03',
  String phone = '',
}) async {
  await enter(tester, 'Given name', given);
  await enter(tester, 'Family name', family);
  await enter(tester, 'Date of birth', birth);
  await enter(tester, 'Phone (optional)', phone);
}

void main() {
  testWidgets('a patient without a profile is asked to complete it after signing in', (tester) async {
    final h = await withoutProfile(tester);

    expect(find.text('Complete your profile'), findsOneWidget);
    expect(h.api.last('GET /me/patient').headers['Authorization'], 'Bearer $token');
  });

  testWidgets('can leave the profile for later and is reminded on the home screen', (tester) async {
    await withoutProfile(tester);
    await tapLink(tester, 'Not now');

    expect(find.text('Signed in as $email'), findsOneWidget);
    expect(find.text('Complete your profile before you book.'), findsOneWidget);

    await tapOutlined(tester, 'Complete your profile');
    expect(find.widgetWithText(TextFormField, 'Given name'), findsOneWidget);
  });

  testWidgets('a profile completed on another screen clears the reminder on return', (tester) async {
    final h = await withoutProfile(tester);
    await tapLink(tester, 'Not now');
    h.api.answer('GET /me/consents', 200, {'consents': []});
    await tapOutlined(tester, 'My consents');

    h.api.answer('GET /me/patient', 200, patient());
    await tester.pageBack();
    await tester.pumpAndSettle();

    expect(find.text('Complete your profile before you book.'), findsNothing);
    expect(find.widgetWithText(OutlinedButton, 'My profile'), findsOneWidget);
  });

  testWidgets('checks every field before calling the API', (tester) async {
    final cases = {
      'no given name': (
        given: '  ',
        family: 'Example',
        birth: '1990-02-03',
        phone: '',
        message: 'Enter your given name, up to 100 characters.',
      ),
      '101 characters': (
        given: 'a' * 101,
        family: 'Example',
        birth: '1990-02-03',
        phone: '',
        message: 'Enter your given name, up to 100 characters.',
      ),
      'no family name': (
        given: 'Ada',
        family: '',
        birth: '1990-02-03',
        phone: '',
        message: 'Enter your family name, up to 100 characters.',
      ),
      'not a date': (
        given: 'Ada',
        family: 'Example',
        birth: '3.2.1990',
        phone: '',
        message: 'Enter your date of birth as YYYY-MM-DD.',
      ),
      'no such day': (
        given: 'Ada',
        family: 'Example',
        birth: '1990-02-30',
        phone: '',
        message: 'Enter your date of birth as YYYY-MM-DD.',
      ),
      'today': (
        given: 'Ada',
        family: 'Example',
        birth: '2026-11-04',
        phone: '',
        message: 'Enter a date of birth in the past, from 1900 on.',
      ),
      'before 1900': (
        given: 'Ada',
        family: 'Example',
        birth: '1899-12-31',
        phone: '',
        message: 'Enter a date of birth in the past, from 1900 on.',
      ),
      'no country code': (
        given: 'Ada',
        family: 'Example',
        birth: '1990-02-03',
        phone: '060 000 0001',
        message: 'Enter the phone number with its country code, for example +387 60 000 0000.',
      ),
      'too short': (
        given: 'Ada',
        family: 'Example',
        birth: '1990-02-03',
        phone: '+387 60',
        message: 'Enter the phone number with its country code, for example +387 60 000 0000.',
      ),
    };
    for (final MapEntry(key: name, value: c) in cases.entries) {
      final h = await withoutProfile(tester);
      await fill(tester, given: c.given, family: c.family, birth: c.birth, phone: c.phone);
      await tapButton(tester, 'Save');

      expect(find.text(c.message), findsOneWidget, reason: name);
      expect(h.api.calls.where((c) => c.startsWith('PUT')), isEmpty, reason: name);
    }
  });

  testWidgets('creates the profile with trimmed names and the phone without spaces, then shows home', (tester) async {
    final h = await withoutProfile(tester);
    h.api.answer('PUT /me/patient', 201, patient());
    await fill(tester, given: ' Ada ', family: 'Example ', birth: '1900-01-01', phone: '+387 60 000 0001');
    await tapButton(tester, 'Save');

    expect(jsonDecode(h.api.last('PUT /me/patient').body), {
      'givenName': 'Ada',
      'familyName': 'Example',
      'birthDate': '1900-01-01',
      'phone': '+387600000001',
      'versionId': null,
    });
    expect(find.text('Signed in as $email'), findsOneWidget);
    expect(find.text('Complete your profile before you book.'), findsNothing);
    expectNoCsrfHeader(h.api);
  });

  testWidgets('an empty phone is sent as none', (tester) async {
    final h = await withoutProfile(tester);
    h.api.answer('PUT /me/patient', 201, patient());
    await fill(tester);
    await tapButton(tester, 'Save');

    expect(jsonDecode(h.api.last('PUT /me/patient').body)['phone'], isNull);
  });

  testWidgets('edits the profile at the version it was read', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.answer('GET /auth/me', 200, account());
    await h.pump(tester);
    await tapOutlined(tester, 'My profile');

    expect(find.widgetWithText(TextFormField, 'Ada'), findsOneWidget);
    expect(find.widgetWithText(TextFormField, '1990-02-03'), findsOneWidget);
    expect(find.widgetWithText(TextFormField, '+387600000001'), findsOneWidget);

    h.api.answer('PUT /me/patient', 200, patient(version: 2));
    await enter(tester, 'Given name', 'Adela');
    await tapButton(tester, 'Save');

    final body = jsonDecode(h.api.last('PUT /me/patient').body);
    expect(body['givenName'], 'Adela');
    expect(body['versionId'], 1);
    expect(find.text('Signed in as $email'), findsOneWidget);
  });

  testWidgets('a profile changed elsewhere is read again and the patient is told to check it', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.answer('GET /auth/me', 200, account());
    await h.pump(tester);
    await tapOutlined(tester, 'My profile');

    h.api.answer('PUT /me/patient', 409, error('version_conflict'));
    h.api.answer('GET /me/patient', 200, {...patient(version: 3), 'givenName': 'Ana'});
    await enter(tester, 'Given name', 'Adela');
    await tapButton(tester, 'Save');

    expect(find.text('Your profile was changed in the meantime. Check it and save again.'), findsOneWidget);
    expect(find.widgetWithText(TextFormField, 'Ana'), findsOneWidget);

    h.api.answer('PUT /me/patient', 200, patient(version: 4));
    await tapButton(tester, 'Save');
    expect(jsonDecode(h.api.last('PUT /me/patient').body)['versionId'], 3);
  });

  testWidgets('the API refusing a field is a fixed message', (tester) async {
    final cases = {
      'invalid_name': 'Enter your given and family name, up to 100 characters each.',
      'invalid_birth_date': 'Enter a date of birth in the past, from 1900 on.',
      'invalid_phone': 'Enter the phone number with its country code, for example +387 60 000 0000.',
      'invalid_request': 'Something went wrong. Try again.',
    };
    for (final MapEntry(key: code, value: message) in cases.entries) {
      final h = await withoutProfile(tester);
      h.api.answer('PUT /me/patient', 400, error(code));
      await fill(tester);
      await tapButton(tester, 'Save');

      expect(find.text(message), findsOneWidget, reason: code);
    }
  });

  testWidgets('a failed profile read offers to try again', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.answer('GET /auth/me', 200, account());
    h.api.answer('GET /me/patient', 500, error('internal error'));
    await h.pump(tester);

    expect(find.text('Something went wrong. Try again.'), findsOneWidget);
    h.api.answer('GET /me/patient', 200, patient());
    await tapButton(tester, 'Try again');
    expect(find.text('Signed in as $email'), findsOneWidget);
  });
}
