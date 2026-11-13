import 'dart:convert';

import 'package:salus_api/api.dart' show ApiClient, QueryParam;

import 'documents.dart';
import 'health_profile.dart';
import 'questionnaire.dart';
import 'session.dart';

// The FHIR reads the app makes, hand-written for the elements it shows (docs/booking/SPEC.md "Web and mobile
// read FHIR"). Writes go through the generated client.

const _zoneExtension = 'https://salus.invalid/fhir/StructureDefinition/organization-timezone';

/// A listed clinic: FHIR Organization with the time-zone extension.
class Organization {
  const Organization({required this.id, required this.name, required this.timeZone});

  final String id;
  final String name;

  /// IANA zone, `''` when the resource has none.
  final String timeZone;
}

/// A practitioner of a clinic: FHIR PractitionerRole with the included Practitioner's name.
class PractitionerRole {
  const PractitionerRole({required this.id, required this.name, this.specialty, this.specialtyCode});

  final String id;
  final String name;

  /// The specialty's text (or display) as the API gives it; null when the role has none.
  final String? specialty;

  /// The specialty's SNOMED CT code, which names it in the app's language; null when the role has none.
  final String? specialtyCode;
}

/// A free FHIR Slot.
class Slot {
  const Slot({required this.id, required this.start, required this.end});

  final String id;
  final DateTime start;
  final DateTime end;
}

/// The patient's FHIR Appointment.
class Appointment {
  const Appointment({
    required this.id,
    required this.status,
    required this.start,
    required this.end,
    this.roleId,
    this.confirmed = false,
  });

  final String id;

  /// FHIR AppointmentStatus: booked, cancelled, fulfilled or noshow.
  final String status;
  final DateTime start;
  final DateTime end;

  /// The PractitionerRole participant's id.
  final String? roleId;

  /// Whether the patient confirmed they will come: their participant is `accepted` (`tentative` until then,
  /// docs/plans/appointments.md Q3).
  final bool confirmed;
}

/// Searches under /fhir/R4 through the generated client's transport, so they carry the session's bearer token.
class Fhir {
  const Fhir(this._client);

  final ApiClient _client;

  Future<List<Organization>> listedClinics() async => [
    for (final r in await _search('Organization', [const QueryParam('listed', 'true')]))
      Organization(id: _string(r['id']), name: _string(r['name']), timeZone: _zone(r['extension'])),
  ];

  Future<List<PractitionerRole>> practitioners(String clinicId) async {
    final all = await _search('PractitionerRole', [
      QueryParam('organization', clinicId),
      const QueryParam('_include', 'PractitionerRole:practitioner'),
    ], includes: true);
    final names = {
      for (final r in all)
        if (r['resourceType'] == 'Practitioner') 'Practitioner/${r['id']}': _humanName(r['name']),
    };
    return [
      for (final r in all)
        if (r['resourceType'] == 'PractitionerRole')
          PractitionerRole(
            id: _string(r['id']),
            name: names[_reference(r['practitioner'])] ?? '',
            specialty: _specialty(r['specialty']),
            specialtyCode: _specialtyCode(r['specialty']),
          ),
    ];
  }

  /// The id of [roleId]'s schedule, or null when the role has none yet.
  Future<String?> scheduleOf(String roleId) async {
    final all = await _search('Schedule', [QueryParam('actor', 'PractitionerRole/$roleId')]);
    return all.isEmpty ? null : _string(all.first['id']);
  }

  /// The free slots of the schedule that start in [from, to).
  Future<List<Slot>> freeSlots(String scheduleId, DateTime from, DateTime to) async => [
    for (final r in await _search('Slot', [
      QueryParam('schedule', 'Schedule/$scheduleId'),
      const QueryParam('status', 'free'),
      QueryParam('start', 'ge${_instant(from)}'),
      QueryParam('start', 'lt${_instant(to)}'),
    ]))
      Slot(id: _string(r['id']), start: _time(r['start']), end: _time(r['end'])),
  ];

  Future<List<Appointment>> appointments(String patientId) async => [
    for (final r in await _search('Appointment', [QueryParam('patient', 'Patient/$patientId')]))
      Appointment(
        id: _string(r['id']),
        status: _string(r['status']),
        start: _time(r['start']),
        end: _time(r['end']),
        roleId: _participant(r['participant'], 'PractitionerRole/'),
        confirmed: _participantStatus(r['participant'], 'Patient/') == 'accepted',
      ),
  ];

  /// The patient's questionnaire responses that are not withdrawn, and the versions they answer by canonical
  /// (`<url>|<version>`), in one search (docs/questionnaire/SPEC.md §2 "FHIR").
  Future<({List<Response> responses, Map<String, Version> versions})> questionnaireResponses(String patientId) async {
    final all = await _search('QuestionnaireResponse', [
      QueryParam('patient', 'Patient/$patientId'),
      const QueryParam('_include', 'QuestionnaireResponse:questionnaire'),
    ], includes: true);
    final versions = {
      for (final r in all)
        if (r['resourceType'] == 'Questionnaire') Version.fromFhir(r).canonical: Version.fromFhir(r),
    };
    return (
      responses: [
        for (final r in all)
          if (r['resourceType'] == 'QuestionnaireResponse' && r['status'] != 'stopped') Response.fromFhir(r),
      ],
      versions: versions,
    );
  }

  /// The version [canonical] (`<url>|<version>`) names, of those the patient's own responses answer; null for none.
  Future<Version?> questionnaireVersion(String canonical) async {
    final i = canonical.lastIndexOf('|');
    if (i < 0) return null;
    final found = await _search('Questionnaire', [
      QueryParam('url', canonical.substring(0, i)),
      QueryParam('version', canonical.substring(i + 1)),
    ]);
    return found.isEmpty ? null : Version.fromFhir(found.first);
  }

  /// One questionnaire response, read again.
  Future<Response> questionnaireResponse(String id) async => Response.fromFhir(await _get('QuestionnaireResponse/$id'));

  /// The id of the appointment's Encounter, or null before the consultation has one.
  Future<String?> encounterOf(String appointmentId) async {
    final all = await _search('Encounter', [QueryParam('appointment', appointmentId)]);
    return all.isEmpty ? null : _string(all.first['id']);
  }

  /// The encounter's signed documents, in the order they were signed (docs/consultation/SPEC.md "Patient documents").
  Future<List<SignedDocument>> signedDocuments(String encounterId) async => [
    for (final r in await _search('DocumentReference', [QueryParam('encounter', encounterId)]))
      SignedDocument.fromFhir(r),
  ];

  /// A signed document's Composition.
  Future<DocumentContent> composition(String id) async => DocumentContent.fromFhir(await _get('Composition/$id'));

  /// The allergies of the patient's profile and of their visits, without the removed ones (SAL-41).
  Future<List<Allergy>> allergies(String patientId) async => [
    for (final r in await _search('AllergyIntolerance', [QueryParam('patient', 'Patient/$patientId')]))
      ?Allergy.fromFhir(r),
  ];

  /// The medications of the patient's profile and of their visits, without the removed ones (SAL-41).
  Future<List<Medication>> medications(String patientId) async => [
    for (final r in await _search('MedicationStatement', [QueryParam('patient', 'Patient/$patientId')]))
      ?Medication.fromFhir(r),
  ];

  /// The resources of a searchset: the matches, and the included ones when [includes].
  Future<List<Map<String, dynamic>>> _search(String type, List<QueryParam> query, {bool includes = false}) async {
    final bundle = await _get(type, query);
    return [
      for (final entry in (bundle['entry'] as List?) ?? const [])
        if (includes || entry['search']?['mode'] != 'include') entry['resource'] as Map<String, dynamic>,
    ];
  }

  /// The resource or bundle at /fhir/R4/[path].
  Future<Map<String, dynamic>> _get(String path, [List<QueryParam> query = const []]) async {
    final response = await apiCall(
      () => _client.invokeAPI('/fhir/R4/$path', 'GET', query, null, {'Accept': 'application/fhir+json'}, {}, null),
    );
    if (response.statusCode >= 400) throw ApiFailure(status: response.statusCode);
    // FHIR JSON is UTF-8 (the media type has no charset parameter, so the body is decoded here, not by http).
    return jsonDecode(utf8.decode(response.bodyBytes)) as Map<String, dynamic>;
  }

  static String _instant(DateTime t) => '${t.toUtc().toIso8601String().split('.').first}Z';

  static String _string(Object? v) => v is String ? v : '';

  static DateTime _time(Object? v) => DateTime.parse(_string(v)).toUtc();

  static String _reference(Object? ref) => ref is Map ? _string(ref['reference']) : '';

  static String _zone(Object? extensions) {
    for (final e in (extensions as List?) ?? const []) {
      if (e['url'] == _zoneExtension) return _string(e['valueCode']);
    }
    return '';
  }

  static String _humanName(Object? names) {
    if (names is! List || names.isEmpty) return '';
    final name = names.first as Map;
    return [...((name['given'] as List?) ?? const []), name['family']].whereType<String>().join(' ');
  }

  static String? _specialty(Object? specialties) {
    if (specialties is! List || specialties.isEmpty) return null;
    final concept = specialties.first as Map;
    final coding = (concept['coding'] as List?)?.firstOrNull as Map?;
    return (concept['text'] ?? coding?['display']) as String?;
  }

  static String? _specialtyCode(Object? specialties) {
    if (specialties is! List || specialties.isEmpty) return null;
    final coding = ((specialties.first as Map)['coding'] as List?)?.firstOrNull as Map?;
    return coding?['system'] == 'http://snomed.info/sct' ? coding!['code'] as String? : null;
  }

  /// The status of the first participant whose actor's reference starts with [prefix].
  static String? _participantStatus(Object? participants, String prefix) {
    for (final p in (participants as List?) ?? const []) {
      if (_reference(p['actor']).startsWith(prefix)) return p['status'] as String?;
    }
    return null;
  }

  static String? _participant(Object? participants, String prefix) {
    for (final p in (participants as List?) ?? const []) {
      final ref = _reference(p['actor']);
      if (ref.startsWith(prefix)) return ref.substring(prefix.length);
    }
    return null;
  }
}
