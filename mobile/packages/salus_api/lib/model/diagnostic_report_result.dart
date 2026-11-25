//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DiagnosticReportResult {
  /// Returns a new [DiagnosticReportResult] instance.
  DiagnosticReportResult({
    required this.test,
    this.loinc,
    this.value,
    this.unit,
    this.valueText,
    this.referenceRange,
    this.interpretation,
  });

  /// The test's name as the lab prints it (Observation.code.text).
  String test;

  /// Its LOINC code (http://loinc.org); absent for none.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? loinc;

  /// Observation.valueQuantity.value.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  num? value;

  /// A UCUM unit (http://unitsofmeasure.org), with value only.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? unit;

  /// A text value instead of a number (Observation.valueString).
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? valueText;

  /// As the lab prints it.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? referenceRange;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  ObservationInterpretation? interpretation;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DiagnosticReportResult &&
          other.test == test &&
          other.loinc == loinc &&
          other.value == value &&
          other.unit == unit &&
          other.valueText == valueText &&
          other.referenceRange == referenceRange &&
          other.interpretation == interpretation;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (test.hashCode) +
      (loinc == null ? 0 : loinc!.hashCode) +
      (value == null ? 0 : value!.hashCode) +
      (unit == null ? 0 : unit!.hashCode) +
      (valueText == null ? 0 : valueText!.hashCode) +
      (referenceRange == null ? 0 : referenceRange!.hashCode) +
      (interpretation == null ? 0 : interpretation!.hashCode);

  @override
  String toString() =>
      'DiagnosticReportResult[test=$test, loinc=$loinc, value=$value, unit=$unit, valueText=$valueText, referenceRange=$referenceRange, interpretation=$interpretation]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'test'] = this.test;
    if (this.loinc != null) {
      json[r'loinc'] = this.loinc;
    } else {
      json[r'loinc'] = null;
    }
    if (this.value != null) {
      json[r'value'] = this.value;
    } else {
      json[r'value'] = null;
    }
    if (this.unit != null) {
      json[r'unit'] = this.unit;
    } else {
      json[r'unit'] = null;
    }
    if (this.valueText != null) {
      json[r'valueText'] = this.valueText;
    } else {
      json[r'valueText'] = null;
    }
    if (this.referenceRange != null) {
      json[r'referenceRange'] = this.referenceRange;
    } else {
      json[r'referenceRange'] = null;
    }
    if (this.interpretation != null) {
      json[r'interpretation'] = this.interpretation;
    } else {
      json[r'interpretation'] = null;
    }
    return json;
  }

  /// Returns a new [DiagnosticReportResult] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static DiagnosticReportResult? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'test'),
            'Required key "DiagnosticReportResult[test]" is missing from JSON.');
        assert(json[r'test'] != null,
            'Required key "DiagnosticReportResult[test]" has a null value in JSON.');
        return true;
      }());

      return DiagnosticReportResult(
        test: mapValueOfType<String>(json, r'test')!,
        loinc: mapValueOfType<String>(json, r'loinc'),
        value: json[r'value'] == null ? null : num.parse('${json[r'value']}'),
        unit: mapValueOfType<String>(json, r'unit'),
        valueText: mapValueOfType<String>(json, r'valueText'),
        referenceRange: mapValueOfType<String>(json, r'referenceRange'),
        interpretation:
            ObservationInterpretation.fromJson(json[r'interpretation']),
      );
    }
    return null;
  }

  static List<DiagnosticReportResult> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DiagnosticReportResult>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DiagnosticReportResult.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, DiagnosticReportResult> mapFromJson(dynamic json) {
    final map = <String, DiagnosticReportResult>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = DiagnosticReportResult.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of DiagnosticReportResult-objects as value to a dart map
  static Map<String, List<DiagnosticReportResult>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<DiagnosticReportResult>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = DiagnosticReportResult.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'test',
  };
}
