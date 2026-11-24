import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

import 'support.dart';

// Synthetic requests only (docs/clinic-cms/WAVE5.md, D6).
const requestId = 'd0000000-0000-4000-8000-0000000000d1';
const otherId = 'd0000000-0000-4000-8000-0000000000d2';
const thirdId = 'd0000000-0000-4000-8000-0000000000d3';

Map<String, Object?> dataRequest({
  String id = requestId,
  String kind = 'access',
  String? clinic,
  String status = 'received',
  String? extendedUntil,
  String? extensionReason,
  String? answerCode,
  int version = 1,
}) => {
  'id': id,
  'kind': kind,
  'target': clinic == null ? 'account' : 'clinic',
  'organizationId': ?clinic,
  'status': status,
  'receivedAt': '2026-11-04T09:00:00Z',
  'dueAt': '2026-12-04T09:00:00Z',
  'extendedUntil': ?extendedUntil,
  'extensionReason': ?extensionReason,
  'answeredAt': ?(answerCode == null ? null : '2026-11-20T09:00:00Z'),
  'answerCode': ?answerCode,
  'filedByPatient': true,
  'version': version,
};

Future<Harness> atPrivacy(WidgetTester tester, List<Map<String, Object?>> requests) async {
  final h = Harness(store: FakeTokenStore(token));
  h.api.answer('GET /auth/me', 200, account());
  answerDirectory(h.api);
  h.api.answer('GET /me/consents', 200, {
    'consents': [consent('care-one')],
  });
  h.api.answer('GET /me/data-requests', 200, requests);
  await h.pump(tester);
  await tapOutlined(tester, 'Privacy');
  return h;
}

void main() {
  testWidgets('lists the requests with whom they went to, their state and due date', (tester) async {
    await atPrivacy(tester, [
      dataRequest(kind: 'erasure', status: 'in_progress'),
      dataRequest(id: otherId, kind: 'rectification', clinic: clinicId),
    ]);

    expect(find.text('Erase my data'), findsOneWidget);
    expect(find.text('To Salus, about your account'), findsOneWidget);
    expect(find.text('In progress'), findsOneWidget);
    expect(find.text('Correct my data'), findsOneWidget);
    expect(find.text('To $clinicName'), findsOneWidget);
    expect(find.text('Received'), findsOneWidget);
    expect(find.text('Received on Wednesday 4 November 2026'), findsNWidgets(2));
    expect(find.text('Answer due by Friday 4 December 2026'), findsNWidgets(2));
  });

  testWidgets('says when there is no request yet, and offers to make one and to close the account', (tester) async {
    await atPrivacy(tester, []);

    expect(find.text('You have not made a request yet.'), findsOneWidget);
    expect(find.widgetWithText(FilledButton, 'Make a request'), findsOneWidget);
    expect(find.widgetWithText(OutlinedButton, 'Close my account'), findsOneWidget);
  });

  testWidgets('an extension says until when and why; an answer says how it ended', (tester) async {
    await atPrivacy(tester, [
      dataRequest(extendedUntil: '2027-02-04T09:00:00Z', extensionReason: 'complex', status: 'in_progress'),
      dataRequest(id: otherId, kind: 'erasure', clinic: clinicId, status: 'refused', answerCode: 'retention_law'),
    ]);

    expect(find.text('Extended to Thursday 4 February 2027, as the request is complex.'), findsOneWidget);
    expect(find.text('Refused'), findsOneWidget);
    expect(find.text('Refused: the law requires this data to be kept.'), findsOneWidget);
    expect(
      find.text('If you disagree with an answer, you can complain to the data protection authority.'),
      findsOneWidget,
    );
    expect(find.text('Answer due by Friday 4 December 2026'), findsNothing);
  });

  testWidgets('files a request to a clinic of the patient, with the kind explained, and reads the list again', (
    tester,
  ) async {
    final h = await atPrivacy(tester, []);
    h.api.answer('POST /me/data-requests', 201, dataRequest(kind: 'portability', clinic: clinicId));
    await tapButton(tester, 'Make a request');

    expect(
      find.text('You receive the same archive, in a standard format (FHIR) another provider can read.'),
      findsOneWidget,
    );
    await tapText(tester, 'Take my data elsewhere');
    await tapText(tester, clinicName);
    h.api.answer('GET /me/data-requests', 200, [dataRequest(kind: 'portability', clinic: clinicId)]);
    await tapButton(tester, 'Send request');

    expect(sent(h.api.last('POST /me/data-requests').body), {
      'kind': 'portability',
      'target': 'clinic',
      'organizationId': clinicId,
    });
    expect(find.text('Your request was received. The answer is due by Friday 4 December 2026.'), findsOneWidget);
    expect(find.text('Take my data elsewhere'), findsOneWidget);
    expect(find.text('To $clinicName'), findsOneWidget);
    expectNoCsrfHeader(h.api);
  });

  testWidgets('a request goes to the Salus account unless a clinic is chosen', (tester) async {
    final h = await atPrivacy(tester, []);
    h.api.answer('POST /me/data-requests', 201, dataRequest(kind: 'objection'));
    await tapButton(tester, 'Make a request');
    await tapText(tester, 'Object to the use of my data');
    await tapButton(tester, 'Send request');

    expect(sent(h.api.last('POST /me/data-requests').body), {'kind': 'objection', 'target': 'account'});
  });

  testWidgets('asks for a kind before sending', (tester) async {
    final h = await atPrivacy(tester, []);
    await tapButton(tester, 'Make a request');
    await tapButton(tester, 'Send request');

    expect(find.text('Choose what you are asking for.'), findsOneWidget);
    expect(h.api.calls, isNot(contains('POST /me/data-requests')));
  });

  testWidgets('a clinic with no record of the patient says so', (tester) async {
    final h = await atPrivacy(tester, []);
    h.api.answer('POST /me/data-requests', 404, error('not_found'));
    await tapButton(tester, 'Make a request');
    await tapText(tester, 'See my data');
    await tapText(tester, clinicName);
    await tapButton(tester, 'Send request');

    expect(find.text('That clinic holds no record of you.'), findsOneWidget);
  });

  testWidgets('withdraws an open request after the confirmation, with the version read', (tester) async {
    final h = await atPrivacy(tester, [dataRequest(version: 3)]);
    h.api.answer('POST /me/data-requests/$requestId/withdraw', 200, dataRequest(status: 'withdrawn', version: 4));
    await tapOutlined(tester, 'Withdraw request');

    expect(find.text('Withdraw your request?'), findsOneWidget);
    await tapLink(tester, 'Keep request');
    expect(h.api.calls.where((c) => c.contains('withdraw')), isEmpty);

    await tapOutlined(tester, 'Withdraw request');
    h.api.answer('GET /me/data-requests', 200, [dataRequest(status: 'withdrawn', version: 4)]);
    await tapButton(tester, 'Withdraw request');

    expect(h.api.last('POST /me/data-requests/$requestId/withdraw').headers['If-Match'], 'W/"3"');
    expect(find.text('Your request was withdrawn.'), findsOneWidget);
    expect(find.text('Withdrawn'), findsOneWidget);
    expect(find.widgetWithText(OutlinedButton, 'Withdraw request'), findsNothing);
  });

  testWidgets('a request closed meanwhile is not withdrawn and says so', (tester) async {
    final h = await atPrivacy(tester, [dataRequest()]);
    h.api.answer('POST /me/data-requests/$requestId/withdraw', 409, error('data_request_closed'));
    await tapOutlined(tester, 'Withdraw request');
    await tapButton(tester, 'Withdraw request');

    expect(find.text('This request is already closed.'), findsOneWidget);
  });

  testWidgets('a request changed meanwhile is read again, not withdrawn', (tester) async {
    final h = await atPrivacy(tester, [dataRequest()]);
    h.api.answer('POST /me/data-requests/$requestId/withdraw', 412, error('version_conflict'));
    await tapOutlined(tester, 'Withdraw request');
    h.api.answer('GET /me/data-requests', 200, [dataRequest(status: 'in_progress', version: 2)]);
    await tapButton(tester, 'Withdraw request');

    expect(find.text('This request has changed meanwhile. The list shows it as it is now.'), findsOneWidget);
    expect(find.text('In progress'), findsOneWidget);
  });

  testWidgets('an extension for many requests at once says so', (tester) async {
    await atPrivacy(tester, [
      dataRequest(kind: 'objection', extendedUntil: '2027-02-04T09:00:00Z', extensionReason: 'many_requests'),
    ]);

    expect(
      find.text('Extended to Thursday 4 February 2027, as many requests arrived at the same time.'),
      findsOneWidget,
    );
  });

  testWidgets('a ready export is downloaded and handed to the device until it expires', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.answer('GET /auth/me', 200, account());
    h.api.answer('GET /me/data-requests', 200, [dataRequest(status: 'fulfilled', answerCode: 'done')]);
    h.api.answer('GET /me/data-requests/$requestId/export', 200, {
      'status': 'ready',
      'expiresAt': '2026-11-27T09:00:00Z',
    });
    final archive = utf8.encode('synthetic zip bytes');
    h.api.on(
      'GET /me/data-requests/$requestId/export/archive',
      (_) async => http.Response.bytes(archive, 200, headers: {'content-type': 'application/zip'}),
    );
    await h.pump(tester);
    await tapOutlined(tester, 'Privacy');

    expect(find.text('Done.'), findsOneWidget);
    expect(find.text('Ready to download until Friday 27 November 2026.'), findsOneWidget);
    await tapButton(tester, 'Download my data');

    expect(h.files.sharedFiles, hasLength(1));
    final (name, bytes, type) = h.files.sharedFiles.single;
    expect(name, 'salus-export-$requestId.zip');
    expect(bytes, archive);
    expect(type, 'application/zip');
  });

  testWidgets('an export being prepared, expired or failed offers no download', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.answer('GET /auth/me', 200, account());
    h.api.answer('GET /me/data-requests', 200, [
      dataRequest(status: 'in_progress'),
      dataRequest(id: otherId, kind: 'portability', status: 'fulfilled', answerCode: 'done'),
      dataRequest(id: thirdId, status: 'in_progress'),
    ]);
    h.api.answer('GET /me/data-requests/$requestId/export', 200, {'status': 'queued'});
    h.api.answer('GET /me/data-requests/$otherId/export', 200, {'status': 'expired'});
    h.api.answer('GET /me/data-requests/$thirdId/export', 200, {'status': 'failed'});
    await h.pump(tester);
    await tapOutlined(tester, 'Privacy');

    expect(find.text('Your data is being prepared. Look here again later.'), findsOneWidget);
    expect(find.text('The download has expired. Make a new request for a fresh copy.'), findsOneWidget);
    expect(find.text('The archive could not be prepared.'), findsOneWidget);
    expect(find.widgetWithText(FilledButton, 'Download my data'), findsNothing);
  });

  testWidgets('only access and portability requests ask for an export', (tester) async {
    final h = await atPrivacy(tester, [dataRequest(kind: 'objection'), dataRequest(id: otherId, status: 'withdrawn')]);

    expect(h.api.calls.where((c) => c.contains('/export')), isEmpty);
  });

  testWidgets('an export that cannot be read leaves the rest of the list', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.answer('GET /auth/me', 200, account());
    h.api.answer('GET /me/data-requests', 200, [dataRequest(status: 'in_progress')]);
    h.api.answer('GET /me/data-requests/$requestId/export', 503, error('unavailable'));
    await h.pump(tester);
    await tapOutlined(tester, 'Privacy');

    expect(find.text('In progress'), findsOneWidget);
    expect(find.widgetWithText(OutlinedButton, 'Close my account'), findsOneWidget);
  });

  group('closing the account', () {
    Future<Harness> atClose(WidgetTester tester) async {
      final h = await atPrivacy(tester, []);
      await tapOutlined(tester, 'Close my account');
      return h;
    }

    testWidgets('says what happens at once, what is erased after 21 days and what stays', (tester) async {
      await atClose(tester);

      expect(find.textContaining('you are signed out on every device'), findsOneWidget);
      expect(find.textContaining('your future appointments are cancelled'), findsOneWidget);
      expect(find.textContaining('your consents are revoked'), findsOneWidget);
      expect(find.textContaining('After 21 days'), findsOneWidget);
      expect(find.textContaining('within 21 days'), findsOneWidget);
      expect(find.textContaining('Your clinical record stays with each clinic'), findsOneWidget);
    });

    testWidgets('asks for the password before closing', (tester) async {
      final h = await atClose(tester);
      await tapButton(tester, 'Close my account');

      expect(find.text('Enter your password.'), findsOneWidget);
      expect(h.api.calls, isNot(contains('POST /me/account/close')));
    });

    testWidgets('a wrong password closes nothing', (tester) async {
      final h = await atClose(tester);
      h.api.answer('POST /me/account/close', 403, error('wrong_credentials'));
      await enter(tester, 'Password', 'not the password');
      await tapButton(tester, 'Close my account');

      expect(find.text('The e-mail or password is wrong.'), findsOneWidget);
      expect(h.store.token, token);
    });

    testWidgets('closes with the password, then signs out and forgets the token', (tester) async {
      final h = await atClose(tester);
      h.api.answer('POST /me/account/close', 204);
      h.api.answer('POST /auth/logout', 401, error('unauthenticated'));
      await enter(tester, 'Password', password);
      await tapButton(tester, 'Close my account');

      expect(sent(h.api.last('POST /me/account/close').body), {'password': password});
      expect(find.text('Your account is closed and you are signed out.'), findsOneWidget);
      await tapButton(tester, 'Done');

      expect(find.text('Sign in to Salus'), findsOneWidget);
      expect(h.store.token, isNull);
      expectNoCsrfHeader(h.api);
    });

    testWidgets('forgets the token after closing even when the sign-out call fails', (tester) async {
      final h = await atClose(tester);
      h.api.answer('POST /me/account/close', 204);
      h.api.answer('POST /auth/logout', 503, error('unavailable'));
      await enter(tester, 'Password', password);
      await tapButton(tester, 'Close my account');
      await tapButton(tester, 'Done');

      expect(find.text('Sign in to Salus'), findsOneWidget);
      expect(h.store.token, isNull);
    });
  });
}
