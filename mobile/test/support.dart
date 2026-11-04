import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:salus/app.dart';
import 'package:salus/session.dart';
import 'package:salus/token_store.dart';
import 'package:salus_api/api.dart';

// Synthetic test data only.
const email = 'patient@example.test';
const password = 'correct horse battery';
const token = 'synthetic-session-token';
const challenge = 'synthetic-challenge';

/// Keeps the token in memory in place of the platform's secure storage.
class FakeTokenStore implements TokenStore {
  FakeTokenStore([this.token]);

  String? token;

  @override
  Future<String?> read() async => token;

  @override
  Future<void> write(String value) async => token = value;

  @override
  Future<void> delete() async => token = null;
}

typedef Respond = Future<http.Response> Function(http.Request request);

/// The API as the generated client sees it: one answer per "METHOD /path", every request recorded.
class FakeApi {
  final routes = <String, Respond>{};
  final requests = <http.Request>[];

  void on(String route, Respond respond) => routes[route] = respond;

  void answer(String route, int status, [Object? body, Map<String, String> headers = const {}]) =>
      on(route, (_) async => jsonResponse(status, body, headers));

  List<String> get calls => [for (final r in requests) '${r.method} ${r.url.path}'];

  http.Request last(String route) => requests.lastWhere((r) => '${r.method} ${r.url.path}' == route);

  Future<http.Response> _handle(http.Request request) {
    requests.add(request);
    final respond = routes['${request.method} ${request.url.path}'];
    if (respond == null) {
      return Future.value(jsonResponse(404, {'error': 'invalid_request'}));
    }
    return respond(request);
  }
}

http.Response jsonResponse(int status, [Object? body, Map<String, String> headers = const {}]) => http.Response(
  body == null ? '' : jsonEncode(body),
  status,
  headers: {'content-type': 'application/json', ...headers},
);

Map<String, Object?> account({String role = 'patient'}) => {
  'id': '0b8f3c1e-6f1a-4c9e-9d2b-3a1f5e7c9d01',
  'email': email,
  'role': role,
  'mfaEnrolled': false,
};

Map<String, Object?> loginDone() => {'next': 'done', 'token': token, 'expiresAt': '2026-12-04T10:00:00Z'};

Map<String, Object?> loginNext(String next) => {'next': next, 'challenge': challenge};

Map<String, Object?> error(String code) => {'error': code};

/// The app wired to [api] and [store], as main.dart wires it to the network and the device.
class Harness {
  Harness({FakeTokenStore? store}) : store = store ?? FakeTokenStore() {
    final bearer = HttpBearerAuth();
    final client = ApiClient(basePath: 'http://api.test', authentication: bearer)..client = MockClient(api._handle);
    session = Session(api: AuthApi(client), bearer: bearer, store: this.store);
  }

  final api = FakeApi();
  final FakeTokenStore store;
  late final Session session;

  Future<void> pump(WidgetTester tester) async {
    await tester.pumpWidget(SalusApp(key: UniqueKey(), session: session));
    await tester.pumpAndSettle();
  }
}

/// No request may carry the web app's CSRF header: without it the API returns the token in the body.
void expectNoCsrfHeader(FakeApi api) {
  for (final request in api.requests) {
    expect(
      request.headers.keys.map((k) => k.toLowerCase()),
      isNot(contains('x-salus-csrf')),
      reason: '${request.method} ${request.url.path}',
    );
  }
}

Future<void> enter(WidgetTester tester, String label, String text) async {
  await tester.enterText(find.widgetWithText(TextFormField, label), text);
}

Future<void> tapButton(WidgetTester tester, String label) async {
  await tester.tap(find.widgetWithText(FilledButton, label));
  await tester.pumpAndSettle();
}

Future<void> tapLink(WidgetTester tester, String label) async {
  await tester.tap(find.widgetWithText(TextButton, label));
  await tester.pumpAndSettle();
}
