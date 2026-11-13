import 'dart:convert';
import 'dart:io';

import 'package:salus_api/api.dart';

import 'session.dart';

// The allergies and current medications of the patient's profile (SAL-41, docs/plans/patient-history.md): read
// as FHIR (fhir.dart), written through the generated client's history calls.

const _confirmedExtension = 'https://salus.invalid/fhir/StructureDefinition/medicationstatement-confirmed';

/// An allergy as the patient's FHIR AllergyIntolerance search gives it.
class Allergy {
  const Allergy({
    required this.id,
    required this.version,
    required this.text,
    required this.clinicalStatus,
    required this.confirmed,
    required this.fromClinic,
    this.category,
    this.reaction,
    this.severity,
  });

  final String id;

  /// meta.versionId, for If-Match.
  final String version;
  final String text;
  final AllergyClinicalStatus clinicalStatus;
  final bool confirmed;

  /// Recorded by a clinic in one of the patient's visits (it has an encounter), not by the patient.
  final bool fromClinic;
  final AllergyCategory? category;
  final String? reaction;
  final AllergySeverity? severity;

  /// Null for a removed entry (entered-in-error), which is shown to no one.
  static Allergy? fromFhir(Map<String, dynamic> r) {
    final verification = _code(r['verificationStatus']);
    if (verification == 'entered-in-error') return null;
    final reaction = (r['reaction'] as List?)?.firstOrNull as Map?;
    final categories = (r['category'] as List?) ?? const [];
    return Allergy(
      id: _string(r['id']),
      version: _string((r['meta'] as Map?)?['versionId']),
      text: _string((r['code'] as Map?)?['text']),
      clinicalStatus: AllergyClinicalStatus.fromJson(_code(r['clinicalStatus'])) ?? AllergyClinicalStatus.active,
      confirmed: verification == 'confirmed',
      fromClinic: r['encounter'] != null,
      category: categories.isEmpty ? null : AllergyCategory.fromJson(categories.first),
      reaction: ((reaction?['manifestation'] as List?)?.firstOrNull as Map?)?['text'] as String?,
      severity: AllergySeverity.fromJson(reaction?['severity']),
    );
  }
}

/// A medication as the patient's FHIR MedicationStatement search gives it.
class Medication {
  const Medication({
    required this.id,
    required this.version,
    required this.text,
    required this.status,
    required this.dosage,
    required this.confirmed,
    required this.fromClinic,
  });

  final String id;
  final String version;
  final String text;
  final MedicationStatus status;
  final Dosage dosage;
  final bool confirmed;

  /// Recorded by a clinic in one of the patient's visits (it has a context), not by the patient.
  final bool fromClinic;

  /// Null for a removed entry (entered-in-error), which is shown to no one.
  static Medication? fromFhir(Map<String, dynamic> r) {
    final status = MedicationStatus.fromJson(r['status']);
    if (status == null) return null;
    final dosage = ((r['dosage'] as List?)?.firstOrNull as Map?) ?? const {};
    final dose = (((dosage['doseAndRate'] as List?)?.firstOrNull as Map?)?['doseQuantity'] as Map?) ?? const {};
    final repeat = ((dosage['timing'] as Map?)?['repeat'] as Map?) ?? const {};
    return Medication(
      id: _string(r['id']),
      version: _string((r['meta'] as Map?)?['versionId']),
      text: _string((r['medicationCodeableConcept'] as Map?)?['text']),
      status: status,
      dosage: Dosage(
        doseValue: dose['value'] as num?,
        doseUnit: DoseUnit.fromJson(dose['code']),
        frequency: repeat['frequency'] as int?,
        period: repeat['period'] as num?,
        periodUnit: PeriodUnit.fromJson(repeat['periodUnit']),
        asNeeded: dosage['asNeededBoolean'] == true ? true : null,
      ),
      confirmed: [
        for (final e in (r['extension'] as List?) ?? const [])
          if (e['url'] == _confirmedExtension && e['valueBoolean'] == true) e,
      ].isNotEmpty,
      fromClinic: r['context'] != null,
    );
  }
}

String _string(Object? v) => v is String ? v : '';

String? _code(Object? concept) =>
    (((concept as Map?)?['coding'] as List?)?.firstOrNull as Map?)?['code'] as String?;

/// A term of the terminology search, with its label in the app's language.
class TermChoice {
  const TermChoice(this.id, this.label);

  final String id;
  final String label;
}

/// The profile's writes and the substance and medication search. Every failure is an [ApiFailure].
class HealthProfile {
  HealthProfile(ApiClient client) : _client = client, _history = HistoryApi(client);

  final ApiClient _client;
  final HistoryApi _history;

  /// The drugs matching [q], each with its label in [language]; a term without one is left out. Called through
  /// the client's transport: the generated searchTerminology writes its `languages` set as `{en}`, not `en`.
  Future<List<TermChoice>> search(String q, String language) async {
    final lang = TermLanguage.fromJson(language)!;
    final response = await apiCall(
      () => _client.invokeAPI(
        '/terminology/search',
        'GET',
        [
          QueryParam('q', q),
          QueryParam('category', TermCategory.drug.toJson()),
          QueryParam('languages', lang.toJson()),
          const QueryParam('limit', '5'),
        ],
        null,
        {},
        {},
        null,
      ),
    );
    if (response.statusCode >= HttpStatus.badRequest) throw ApiFailure.fromBody(response.statusCode, response.body);
    final list = TermList.fromJson(jsonDecode(utf8.decode(response.bodyBytes)));
    return [
      for (final term in list?.terms ?? const <Term>[])
        for (final label in term.labels.where((l) => l.language == lang).take(1))
          if (label.preferred.length <= 200) TermChoice(term.id, label.preferred),
    ];
  }

  Future<void> addAllergy(AllergyIntoleranceInput input) => apiCall(() => _history.addMyAllergy(input));

  Future<void> updateAllergy(String id, String version, AllergyIntoleranceInput input) =>
      apiCall(() => _history.updateAllergy(_etag(version), id, input));

  Future<void> removeAllergy(String id, String version) => apiCall(() => _history.removeAllergy(_etag(version), id));

  Future<void> allergyToProfile(String id, String version) =>
      apiCall(() => _history.addAllergyToProfile(_etag(version), id));

  Future<void> addMedication(MedicationStatementInput input) => apiCall(() => _history.addMyMedication(input));

  Future<void> updateMedication(String id, String version, MedicationStatementInput input) =>
      apiCall(() => _history.updateMedication(_etag(version), id, input));

  Future<void> removeMedication(String id, String version) =>
      apiCall(() => _history.removeMedication(_etag(version), id));

  Future<void> medicationToProfile(String id, String version) =>
      apiCall(() => _history.addMedicationToProfile(_etag(version), id));

  static String _etag(String version) => 'W/"$version"';
}
