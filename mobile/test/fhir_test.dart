import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:salus/fhir.dart';
import 'package:salus/session.dart';

import 'support.dart';

void main() {
  late Harness h;
  late Fhir fhir;

  setUp(() {
    h = Harness();
    h.bearer.accessToken = token;
    fhir = Fhir(h.client);
  });

  Map<String, List<String>> query(String route) => h.api.last(route).url.queryParametersAll;

  test('lists the listed clinics with their time zone, decoding UTF-8 without a charset', () async {
    h.api.on(
      'GET /fhir/R4/Organization',
      (_) async => fhirResponse(
        200,
        bundle([organization(), organization(id: 'other', name: 'Plain', timeZone: 'Europe/Berlin')]),
      ),
    );

    final clinics = await fhir.listedClinics();

    expect(query('GET /fhir/R4/Organization'), {
      'listed': ['true'],
    });
    expect(
      [for (final c in clinics) (c.id, c.name, c.timeZone)],
      [(clinicId, clinicName, zone), ('other', 'Plain', 'Europe/Berlin')],
    );
    expect(h.api.last('GET /fhir/R4/Organization').headers['Accept'], 'application/fhir+json');
    expect(h.api.last('GET /fhir/R4/Organization').headers['Authorization'], 'Bearer $token');
  });

  test("lists a clinic's practitioners with their name and specialty, as the API gives them", () async {
    h.api.on(
      'GET /fhir/R4/PractitionerRole',
      (_) async => fhirResponse(
        200,
        bundle(
          [practitionerRole(), practitionerRole(id: 'role-2', practitioner: 'pr-2', specialty: null)],
          included: [
            practitioner(),
            practitioner(id: 'pr-2', given: 'Dee', family: 'Sample'),
          ],
        ),
      ),
    );

    final roles = await fhir.practitioners(clinicId);

    expect(query('GET /fhir/R4/PractitionerRole'), {
      'organization': [clinicId],
      '_include': ['PractitionerRole:practitioner'],
    });
    expect(
      [for (final r in roles) (r.id, r.name, r.specialty)],
      [(roleId, 'Cy Example', 'Cardiology'), ('role-2', 'Dee Sample', null)],
    );
  });

  test("finds a role's schedule, or none", () async {
    h.api.on('GET /fhir/R4/Schedule', (_) async => fhirResponse(200, bundle([schedule()])));
    expect(await fhir.scheduleOf(roleId), scheduleId);
    expect(query('GET /fhir/R4/Schedule'), {
      'actor': ['PractitionerRole/$roleId'],
    });

    h.api.on('GET /fhir/R4/Schedule', (_) async => fhirResponse(200, bundle([])));
    expect(await fhir.scheduleOf(roleId), isNull);
  });

  test('searches free slots between two instants, sent in UTC', () async {
    h.api.on(
      'GET /fhir/R4/Slot',
      (_) async => fhirResponse(200, bundle([slot('s1', '2026-11-05T08:00:00Z', '2026-11-05T08:30:00Z')])),
    );

    final slots = await fhir.freeSlots(scheduleId, DateTime.utc(2026, 11, 4, 23), DateTime.utc(2026, 11, 5, 23));

    expect(query('GET /fhir/R4/Slot'), {
      'schedule': ['Schedule/$scheduleId'],
      'status': ['free'],
      'start': ['ge2026-11-04T23:00:00Z', 'lt2026-11-05T23:00:00Z'],
    });
    expect(
      [for (final s in slots) (s.id, s.start, s.end)],
      [('s1', DateTime.utc(2026, 11, 5, 8), DateTime.utc(2026, 11, 5, 8, 30))],
    );
  });

  test("searches the patient's appointments with their role", () async {
    h.api.on(
      'GET /fhir/R4/Appointment',
      (_) async => fhirResponse(
        200,
        bundle([appointment('e1', '2026-11-06T08:00:00Z', '2026-11-06T08:30:00Z', status: 'cancelled')]),
      ),
    );

    final all = await fhir.appointments(patientId);

    expect(query('GET /fhir/R4/Appointment'), {
      'patient': ['Patient/$patientId'],
    });
    expect(
      [for (final a in all) (a.id, a.status, a.start, a.end, a.roleId)],
      [('e1', 'cancelled', DateTime.utc(2026, 11, 6, 8), DateTime.utc(2026, 11, 6, 8, 30), roleId)],
    );
  });

  test('a refused search is a failure with its status and no code; a lost connection is unreachable', () async {
    h.api.on('GET /fhir/R4/Slot', (_) async => fhirResponse(400, {'resourceType': 'OperationOutcome', 'issue': []}));
    await expectLater(
      fhir.freeSlots(scheduleId, DateTime.utc(2026), DateTime.utc(2027)),
      throwsA(isA<ApiFailure>().having((f) => f.status, 'status', 400).having((f) => f.code, 'code', isNull)),
    );

    h.api.on('GET /fhir/R4/Slot', (_) async => throw http.ClientException('network'));
    await expectLater(
      fhir.freeSlots(scheduleId, DateTime.utc(2026), DateTime.utc(2027)),
      throwsA(isA<ApiFailure>().having((f) => f.unreachable, 'unreachable', true)),
    );
  });
}
