//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ClinicDiagnosesRowsInner {
  /// Returns a new [ClinicDiagnosesRowsInner] instance.
  ClinicDiagnosesRowsInner({
    required this.key,
    required this.count,
  });

  String key;

  int count;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ClinicDiagnosesRowsInner &&
          other.key == key &&
          other.count == count;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (key.hashCode) + (count.hashCode);

  @override
  String toString() => 'ClinicDiagnosesRowsInner[key=$key, count=$count]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'key'] = this.key;
    json[r'count'] = this.count;
    return json;
  }

  /// Returns a new [ClinicDiagnosesRowsInner] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ClinicDiagnosesRowsInner? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'key'),
            'Required key "ClinicDiagnosesRowsInner[key]" is missing from JSON.');
        assert(json[r'key'] != null,
            'Required key "ClinicDiagnosesRowsInner[key]" has a null value in JSON.');
        assert(json.containsKey(r'count'),
            'Required key "ClinicDiagnosesRowsInner[count]" is missing from JSON.');
        assert(json[r'count'] != null,
            'Required key "ClinicDiagnosesRowsInner[count]" has a null value in JSON.');
        return true;
      }());

      return ClinicDiagnosesRowsInner(
        key: mapValueOfType<String>(json, r'key')!,
        count: mapValueOfType<int>(json, r'count')!,
      );
    }
    return null;
  }

  static List<ClinicDiagnosesRowsInner> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ClinicDiagnosesRowsInner>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ClinicDiagnosesRowsInner.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ClinicDiagnosesRowsInner> mapFromJson(dynamic json) {
    final map = <String, ClinicDiagnosesRowsInner>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ClinicDiagnosesRowsInner.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ClinicDiagnosesRowsInner-objects as value to a dart map
  static Map<String, List<ClinicDiagnosesRowsInner>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ClinicDiagnosesRowsInner>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ClinicDiagnosesRowsInner.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'key',
    'count',
  };
}
