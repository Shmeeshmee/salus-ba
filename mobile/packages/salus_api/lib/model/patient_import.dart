//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class PatientImport {
  /// Returns a new [PatientImport] instance.
  PatientImport({
    required this.saved,
    this.rows = const [],
  });

  /// Every row was stored; false for a preview or when any row has an error.
  bool saved;

  List<PatientImportRow> rows;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is PatientImport &&
          other.saved == saved &&
          _deepEquality.equals(other.rows, rows);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (saved.hashCode) + (rows.hashCode);

  @override
  String toString() => 'PatientImport[saved=$saved, rows=$rows]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'saved'] = this.saved;
    json[r'rows'] = this.rows;
    return json;
  }

  /// Returns a new [PatientImport] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static PatientImport? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'saved'),
            'Required key "PatientImport[saved]" is missing from JSON.');
        assert(json[r'saved'] != null,
            'Required key "PatientImport[saved]" has a null value in JSON.');
        assert(json.containsKey(r'rows'),
            'Required key "PatientImport[rows]" is missing from JSON.');
        assert(json[r'rows'] != null,
            'Required key "PatientImport[rows]" has a null value in JSON.');
        return true;
      }());

      return PatientImport(
        saved: mapValueOfType<bool>(json, r'saved')!,
        rows: PatientImportRow.listFromJson(json[r'rows']),
      );
    }
    return null;
  }

  static List<PatientImport> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <PatientImport>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = PatientImport.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, PatientImport> mapFromJson(dynamic json) {
    final map = <String, PatientImport>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = PatientImport.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of PatientImport-objects as value to a dart map
  static Map<String, List<PatientImport>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<PatientImport>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = PatientImport.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'saved',
    'rows',
  };
}
