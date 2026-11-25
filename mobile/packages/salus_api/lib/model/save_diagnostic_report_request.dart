//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SaveDiagnosticReportRequest {
  /// Returns a new [SaveDiagnosticReportRequest] instance.
  SaveDiagnosticReportRequest({
    required this.title,
    this.loinc,
    this.effectiveDate,
    this.results = const [],
  });

  String title;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? loinc;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? effectiveDate;

  List<DiagnosticReportResult> results;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SaveDiagnosticReportRequest &&
          other.title == title &&
          other.loinc == loinc &&
          other.effectiveDate == effectiveDate &&
          _deepEquality.equals(other.results, results);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (title.hashCode) +
      (loinc == null ? 0 : loinc!.hashCode) +
      (effectiveDate == null ? 0 : effectiveDate!.hashCode) +
      (results.hashCode);

  @override
  String toString() =>
      'SaveDiagnosticReportRequest[title=$title, loinc=$loinc, effectiveDate=$effectiveDate, results=$results]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
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
    return json;
  }

  /// Returns a new [SaveDiagnosticReportRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SaveDiagnosticReportRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'title'),
            'Required key "SaveDiagnosticReportRequest[title]" is missing from JSON.');
        assert(json[r'title'] != null,
            'Required key "SaveDiagnosticReportRequest[title]" has a null value in JSON.');
        assert(json.containsKey(r'results'),
            'Required key "SaveDiagnosticReportRequest[results]" is missing from JSON.');
        assert(json[r'results'] != null,
            'Required key "SaveDiagnosticReportRequest[results]" has a null value in JSON.');
        return true;
      }());

      return SaveDiagnosticReportRequest(
        title: mapValueOfType<String>(json, r'title')!,
        loinc: mapValueOfType<String>(json, r'loinc'),
        effectiveDate: mapDateTime(json, r'effectiveDate', r''),
        results: DiagnosticReportResult.listFromJson(json[r'results']),
      );
    }
    return null;
  }

  static List<SaveDiagnosticReportRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SaveDiagnosticReportRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SaveDiagnosticReportRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SaveDiagnosticReportRequest> mapFromJson(dynamic json) {
    final map = <String, SaveDiagnosticReportRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SaveDiagnosticReportRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SaveDiagnosticReportRequest-objects as value to a dart map
  static Map<String, List<SaveDiagnosticReportRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SaveDiagnosticReportRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SaveDiagnosticReportRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'title',
    'results',
  };
}
