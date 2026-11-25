//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DiagnosticReport {
  /// Returns a new [DiagnosticReport] instance.
  DiagnosticReport({
    required this.id,
    required this.organizationId,
    required this.patientId,
    required this.appointmentId,
    required this.status,
    required this.title,
    this.loinc,
    this.effectiveDate,
    this.results = const [],
    required this.hasPdf,
    required this.createdAt,
    this.releasedAt,
    required this.versionId,
  });

  String id;

  String organizationId;

  String patientId;

  String appointmentId;

  DiagnosticReportStatus status;

  /// The report's name (DiagnosticReport.code.text).
  String title;

  /// Its LOINC code; absent for none.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? loinc;

  /// The sample's day, as the lab says; absent if it does not.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? effectiveDate;

  List<DiagnosticReportResult> results;

  /// Whether the lab's PDF is attached.
  bool hasPdf;

  DateTime createdAt;

  /// Absent until released.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? releasedAt;

  int versionId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DiagnosticReport &&
          other.id == id &&
          other.organizationId == organizationId &&
          other.patientId == patientId &&
          other.appointmentId == appointmentId &&
          other.status == status &&
          other.title == title &&
          other.loinc == loinc &&
          other.effectiveDate == effectiveDate &&
          _deepEquality.equals(other.results, results) &&
          other.hasPdf == hasPdf &&
          other.createdAt == createdAt &&
          other.releasedAt == releasedAt &&
          other.versionId == versionId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (organizationId.hashCode) +
      (patientId.hashCode) +
      (appointmentId.hashCode) +
      (status.hashCode) +
      (title.hashCode) +
      (loinc == null ? 0 : loinc!.hashCode) +
      (effectiveDate == null ? 0 : effectiveDate!.hashCode) +
      (results.hashCode) +
      (hasPdf.hashCode) +
      (createdAt.hashCode) +
      (releasedAt == null ? 0 : releasedAt!.hashCode) +
      (versionId.hashCode);

  @override
  String toString() =>
      'DiagnosticReport[id=$id, organizationId=$organizationId, patientId=$patientId, appointmentId=$appointmentId, status=$status, title=$title, loinc=$loinc, effectiveDate=$effectiveDate, results=$results, hasPdf=$hasPdf, createdAt=$createdAt, releasedAt=$releasedAt, versionId=$versionId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'organizationId'] = this.organizationId;
    json[r'patientId'] = this.patientId;
    json[r'appointmentId'] = this.appointmentId;
    json[r'status'] = this.status;
    json[r'title'] = this.title;
    if (this.loinc != null) {
      json[r'loinc'] = this.loinc;
    } else {
      json[r'loinc'] = null;
    }
    if (this.effectiveDate != null) {
      json[r'effectiveDate'] = _dateFormatter.format(this.effectiveDate!);
    } else {
      json[r'effectiveDate'] = null;
    }
    json[r'results'] = this.results;
    json[r'hasPdf'] = this.hasPdf;
    json[r'createdAt'] = this.createdAt.toUtc().toIso8601String();
    if (this.releasedAt != null) {
      json[r'releasedAt'] = this.releasedAt!.toUtc().toIso8601String();
    } else {
      json[r'releasedAt'] = null;
    }
    json[r'versionId'] = this.versionId;
    return json;
  }

  /// Returns a new [DiagnosticReport] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static DiagnosticReport? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "DiagnosticReport[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "DiagnosticReport[id]" has a null value in JSON.');
        assert(json.containsKey(r'organizationId'),
            'Required key "DiagnosticReport[organizationId]" is missing from JSON.');
        assert(json[r'organizationId'] != null,
            'Required key "DiagnosticReport[organizationId]" has a null value in JSON.');
        assert(json.containsKey(r'patientId'),
            'Required key "DiagnosticReport[patientId]" is missing from JSON.');
        assert(json[r'patientId'] != null,
            'Required key "DiagnosticReport[patientId]" has a null value in JSON.');
        assert(json.containsKey(r'appointmentId'),
            'Required key "DiagnosticReport[appointmentId]" is missing from JSON.');
        assert(json[r'appointmentId'] != null,
            'Required key "DiagnosticReport[appointmentId]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "DiagnosticReport[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "DiagnosticReport[status]" has a null value in JSON.');
        assert(json.containsKey(r'title'),
            'Required key "DiagnosticReport[title]" is missing from JSON.');
        assert(json[r'title'] != null,
            'Required key "DiagnosticReport[title]" has a null value in JSON.');
        assert(json.containsKey(r'results'),
            'Required key "DiagnosticReport[results]" is missing from JSON.');
        assert(json[r'results'] != null,
            'Required key "DiagnosticReport[results]" has a null value in JSON.');
        assert(json.containsKey(r'hasPdf'),
            'Required key "DiagnosticReport[hasPdf]" is missing from JSON.');
        assert(json[r'hasPdf'] != null,
            'Required key "DiagnosticReport[hasPdf]" has a null value in JSON.');
        assert(json.containsKey(r'createdAt'),
            'Required key "DiagnosticReport[createdAt]" is missing from JSON.');
        assert(json[r'createdAt'] != null,
            'Required key "DiagnosticReport[createdAt]" has a null value in JSON.');
        assert(json.containsKey(r'versionId'),
            'Required key "DiagnosticReport[versionId]" is missing from JSON.');
        assert(json[r'versionId'] != null,
            'Required key "DiagnosticReport[versionId]" has a null value in JSON.');
        return true;
      }());

      return DiagnosticReport(
        id: mapValueOfType<String>(json, r'id')!,
        organizationId: mapValueOfType<String>(json, r'organizationId')!,
        patientId: mapValueOfType<String>(json, r'patientId')!,
        appointmentId: mapValueOfType<String>(json, r'appointmentId')!,
        status: DiagnosticReportStatus.fromJson(json[r'status'])!,
        title: mapValueOfType<String>(json, r'title')!,
        loinc: mapValueOfType<String>(json, r'loinc'),
        effectiveDate: mapDateTime(json, r'effectiveDate', r''),
        results: DiagnosticReportResult.listFromJson(json[r'results']),
        hasPdf: mapValueOfType<bool>(json, r'hasPdf')!,
        createdAt: mapDateTime(json, r'createdAt', r'')!,
        releasedAt: mapDateTime(json, r'releasedAt', r''),
        versionId: mapValueOfType<int>(json, r'versionId')!,
      );
    }
    return null;
  }

  static List<DiagnosticReport> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DiagnosticReport>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DiagnosticReport.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, DiagnosticReport> mapFromJson(dynamic json) {
    final map = <String, DiagnosticReport>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = DiagnosticReport.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of DiagnosticReport-objects as value to a dart map
  static Map<String, List<DiagnosticReport>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<DiagnosticReport>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = DiagnosticReport.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'id',
    'organizationId',
    'patientId',
    'appointmentId',
    'status',
    'title',
    'results',
    'hasPdf',
    'createdAt',
    'versionId',
  };
}
