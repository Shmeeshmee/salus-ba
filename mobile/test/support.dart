import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:salus/app.dart';
import 'package:salus/booking.dart';
import 'package:salus/session.dart';
import 'package:salus/token_store.dart';
import 'package:salus_api/api.dart';

import 'fake_call.dart';

// Synthetic test data only.
const email = 'patient@example.test';
const password = 'correct horse battery';
const token = 'synthetic-session-token';
const challenge = 'synthetic-challenge';

/// The clock of every widget test: Wednesday 4 November 2026, 10:00 in Europe/Sarajevo.
final wednesday = DateTime.utc(2026, 11, 4, 9);

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

/// A FHIR answer as the API sends it: UTF-8 bytes, `application/fhir+json` without a charset.
http.Response fhirResponse(int status, Object body) =>
    http.Response.bytes(utf8.encode(jsonEncode(body)), status, headers: {'content-type': 'application/fhir+json'});

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

/// The privacy notice's address in every widget test (PRIVACY_NOTICE_URL).
final privacyNoticeUrl = Uri.parse('https://privacy.example.test/notice');

/// The app wired to [api] and [store], as main.dart wires it to the network and the device.
class Harness {
  /// A signed-in patient with a profile unless the test answers `GET /me/patient` otherwise.
  Harness({FakeTokenStore? store, DateTime? now}) : store = store ?? FakeTokenStore(), now = now ?? wednesday {
    client = ApiClient(basePath: 'http://api.test', authentication: bearer)..client = MockClient(api._handle);
    session = Session(api: AuthApi(client), bearer: bearer, store: this.store);
    booking = Booking(client);
    api.answer('GET /me/patient', 200, patient());
  }

  final api = FakeApi();

  /// The links the app asked the device to open, and whether opening one succeeds.
  final opened = <Uri>[];
  var openSucceeds = true;
  final FakeTokenStore store;
  final bearer = HttpBearerAuth();

  /// The camera, the microphone and LiveKit.
  final media = FakeMedia();

  /// The clock the app reads; a test moves it.
  DateTime now;
  late final ApiClient client;
  late final Session session;
  late final Booking booking;

  Future<void> pump(WidgetTester tester) async {
    await tester.pumpWidget(
      SalusApp(
        key: UniqueKey(),
        session: session,
        booking: booking,
        now: () => now,
        media: media,
        privacyNoticeUrl: privacyNoticeUrl,
        openUrl: (url) async {
          opened.add(url);
          return openSucceeds;
        },
      ),
    );
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
  final field = find.widgetWithText(TextFormField, label);
  await tester.ensureVisible(field);
  await tester.enterText(field, text);
}

/// Scrolls [finder] into view, taps it and waits for what follows.
Future<void> tapFound(WidgetTester tester, Finder finder) async {
  await tester.ensureVisible(finder);
  await tester.pumpAndSettle();
  await tester.tap(finder);
  await tester.pumpAndSettle();
}

Future<void> tapButton(WidgetTester tester, String label) => tapFound(tester, find.widgetWithText(FilledButton, label));

Future<void> tapLink(WidgetTester tester, String label) => tapFound(tester, find.widgetWithText(TextButton, label));

// Synthetic FHIR resources. The ids are opaque; the names are invented.
const clinicId = 'a0000000-0000-4000-8000-00000000000a';
const clinicName = 'Synthetic clinic Čaršija';
const zone = 'Europe/Sarajevo';
const practitionerId = 'a3000000-0000-4000-8000-0000000000a1';
const roleId = 'a4000000-0000-4000-8000-0000000000a1';
const scheduleId = 'a5000000-0000-4000-8000-0000000000a1';
const patientId = 'aa000000-0000-4000-8000-0000000000aa';
const listedUrl = 'https://salus.invalid/fhir/StructureDefinition/organization-listed';
const zoneUrl = 'https://salus.invalid/fhir/StructureDefinition/organization-timezone';

Map<String, Object?> bundle(List<Map<String, Object?>> resources, {List<Map<String, Object?>> included = const []}) => {
  'resourceType': 'Bundle',
  'type': 'searchset',
  'entry': [
    for (final r in resources)
      {
        'fullUrl': 'http://api.test/fhir/R4/${r['resourceType']}/${r['id']}',
        'resource': r,
        'search': {'mode': 'match'},
      },
    for (final r in included)
      {
        'fullUrl': 'http://api.test/fhir/R4/${r['resourceType']}/${r['id']}',
        'resource': r,
        'search': {'mode': 'include'},
      },
  ],
  'total': resources.length,
};

Map<String, Object?> organization({String id = clinicId, String name = clinicName, String timeZone = zone}) => {
  'resourceType': 'Organization',
  'id': id,
  'name': name,
  'meta': {'lastUpdated': '2026-11-04T09:30:00Z', 'versionId': '4'},
  'extension': [
    {'url': listedUrl, 'valueBoolean': true},
    {'url': zoneUrl, 'valueCode': timeZone},
  ],
  'active': true,
};

Map<String, Object?> practitionerRole({
  String id = roleId,
  String practitioner = practitionerId,
  String org = clinicId,
  String? specialty = 'Cardiology',
}) => {
  'resourceType': 'PractitionerRole',
  'id': id,
  'practitioner': {'reference': 'Practitioner/$practitioner'},
  'organization': {'reference': 'Organization/$org'},
  'meta': {'lastUpdated': '2026-11-04T09:30:00Z', 'versionId': '1'},
  if (specialty != null)
    'specialty': [
      {
        'text': specialty,
        'coding': [
          {'system': 'http://snomed.info/sct', 'code': '394579002', 'display': 'Cardiology'},
        ],
      },
    ],
  'active': true,
};

Map<String, Object?> practitioner({String id = practitionerId, String given = 'Cy', String family = 'Example'}) => {
  'resourceType': 'Practitioner',
  'id': id,
  'meta': {'lastUpdated': '2026-11-04T09:30:00Z', 'versionId': '1'},
  'name': [
    {
      'family': family,
      'given': [given],
    },
  ],
  'active': true,
};

Map<String, Object?> schedule({String id = scheduleId, String role = roleId}) => {
  'resourceType': 'Schedule',
  'id': id,
  'meta': {'lastUpdated': '2026-11-04T09:30:00Z', 'versionId': '2'},
  'actor': [
    {'reference': 'PractitionerRole/$role'},
  ],
  'active': true,
};

Map<String, Object?> slot(String id, String start, String end) => {
  'resourceType': 'Slot',
  'id': id,
  'status': 'free',
  'start': start,
  'end': end,
  'schedule': {'reference': 'Schedule/$scheduleId'},
  'meta': {'lastUpdated': '2026-11-04T09:30:00Z', 'versionId': '1'},
};

Map<String, Object?> appointment(
  String id,
  String start,
  String end, {
  String status = 'booked',
  String role = roleId,
  String practitioner = practitionerId,
}) => {
  'resourceType': 'Appointment',
  'id': id,
  'status': status,
  'start': start,
  'end': end,
  'created': '2026-11-01T08:00:00Z',
  'meta': {'lastUpdated': '2026-11-01T08:00:00Z', 'versionId': '1'},
  'participant': [
    {
      'status': 'accepted',
      'actor': {'reference': 'Patient/$patientId', 'display': 'Ada Example'},
    },
    {
      'status': 'accepted',
      'actor': {'reference': 'Practitioner/$practitioner'},
    },
    {
      'status': 'accepted',
      'actor': {'reference': 'PractitionerRole/$role'},
    },
  ],
};

/// A synthetic profile: invented name, a phone number from the fake +387 60 000 0xxx range.
Map<String, Object?> patient({int version = 1}) => {
  'id': patientId,
  'givenName': 'Ada',
  'familyName': 'Example',
  'birthDate': '1990-02-03',
  'phone': '+387600000001',
  'versionId': version,
  'lastUpdated': '2026-11-01T08:00:00Z',
};

Map<String, Object?> consent(String id, {String org = clinicId, String? revokedAt}) => {
  'id': id,
  'organizationId': org,
  'status': revokedAt == null ? 'active' : 'inactive',
  'dateTime': '2026-11-01T08:00:00Z',
  'revokedAt': ?revokedAt,
};

/// The JSON answer of a booking or a cancellation.
Map<String, Object?> booked(String id, {String status = 'booked'}) => {
  'id': id,
  'organizationId': clinicId,
  'practitionerRoleId': roleId,
  'slotId': 'a6000000-0000-4000-8000-0000000000a1',
  'start': '2026-11-05T08:00:00Z',
  'end': '2026-11-05T08:30:00Z',
  'status': status,
  'versionId': 1,
};

/// Answers the directory searches for the one synthetic clinic and practitioner.
void answerDirectory(FakeApi api) {
  api.on('GET /fhir/R4/Organization', (_) async => fhirResponse(200, bundle([organization()])));
  api.on(
    'GET /fhir/R4/PractitionerRole',
    (_) async => fhirResponse(200, bundle([practitionerRole()], included: [practitioner()])),
  );
}

Future<void> tapOutlined(WidgetTester tester, String label) =>
    tapFound(tester, find.widgetWithText(OutlinedButton, label));

Future<void> tapText(WidgetTester tester, String text) => tapFound(tester, find.text(text));
