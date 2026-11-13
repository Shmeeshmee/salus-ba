//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class HistoryDiagnosis {
  /// Returns a new [HistoryDiagnosis] instance.
  HistoryDiagnosis({
    required this.code,
    required this.label,
    required this.confirmed,
  });

  /// ICD-10 (http://hl7.org/fhir/sid/icd-10)
  String code;

  /// The field's label in the document's template.
  String label;

  /// Whether its Condition is confirmed.
  bool confirmed;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is HistoryDiagnosis &&
          other.code == code &&
          other.label == label &&
          other.confirmed == confirmed;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (code.hashCode) + (label.hashCode) + (confirmed.hashCode);

  @override
  String toString() =>
      'HistoryDiagnosis[code=$code, label=$label, confirmed=$confirmed]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'code'] = this.code;
    json[r'label'] = this.label;
    json[r'confirmed'] = this.confirmed;
    return json;
  }

  /// Returns a new [HistoryDiagnosis] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static HistoryDiagnosis? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'code'),
            'Required key "HistoryDiagnosis[code]" is missing from JSON.');
        assert(json[r'code'] != null,
            'Required key "HistoryDiagnosis[code]" has a null value in JSON.');
        assert(json.containsKey(r'label'),
            'Required key "HistoryDiagnosis[label]" is missing from JSON.');
        assert(json[r'label'] != null,
            'Required key "HistoryDiagnosis[label]" has a null value in JSON.');
        assert(json.containsKey(r'confirmed'),
            'Required key "HistoryDiagnosis[confirmed]" is missing from JSON.');
        assert(json[r'confirmed'] != null,
            'Required key "HistoryDiagnosis[confirmed]" has a null value in JSON.');
        return true;
      }());

      return HistoryDiagnosis(
        code: mapValueOfType<String>(json, r'code')!,
        label: mapValueOfType<String>(json, r'label')!,
        confirmed: mapValueOfType<bool>(json, r'confirmed')!,
      );
    }
    return null;
  }

  static List<HistoryDiagnosis> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <HistoryDiagnosis>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = HistoryDiagnosis.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, HistoryDiagnosis> mapFromJson(dynamic json) {
    final map = <String, HistoryDiagnosis>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = HistoryDiagnosis.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of HistoryDiagnosis-objects as value to a dart map
  static Map<String, List<HistoryDiagnosis>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<HistoryDiagnosis>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = HistoryDiagnosis.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'code',
    'label',
    'confirmed',
  };
}
