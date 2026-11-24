//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ImportedPatient {
  /// Returns a new [ImportedPatient] instance.
  ImportedPatient({
    this.id,
    required this.givenName,
    required this.familyName,
    required this.birthDate,
    this.phone,
  });

  /// The profile's id; only once saved.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? id;

  String givenName;

  String familyName;

  DateTime birthDate;

  /// E.164.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? phone;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ImportedPatient &&
          other.id == id &&
          other.givenName == givenName &&
          other.familyName == familyName &&
          other.birthDate == birthDate &&
          other.phone == phone;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id == null ? 0 : id!.hashCode) +
      (givenName.hashCode) +
      (familyName.hashCode) +
      (birthDate.hashCode) +
      (phone == null ? 0 : phone!.hashCode);

  @override
  String toString() =>
      'ImportedPatient[id=$id, givenName=$givenName, familyName=$familyName, birthDate=$birthDate, phone=$phone]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    if (this.id != null) {
      json[r'id'] = this.id;
    } else {
      json[r'id'] = null;
    }
    json[r'givenName'] = this.givenName;
    json[r'familyName'] = this.familyName;
    json[r'birthDate'] = _dateFormatter.format(this.birthDate);
    if (this.phone != null) {
      json[r'phone'] = this.phone;
    } else {
      json[r'phone'] = null;
    }
    return json;
  }

  /// Returns a new [ImportedPatient] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ImportedPatient? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'givenName'),
            'Required key "ImportedPatient[givenName]" is missing from JSON.');
        assert(json[r'givenName'] != null,
            'Required key "ImportedPatient[givenName]" has a null value in JSON.');
        assert(json.containsKey(r'familyName'),
            'Required key "ImportedPatient[familyName]" is missing from JSON.');
        assert(json[r'familyName'] != null,
            'Required key "ImportedPatient[familyName]" has a null value in JSON.');
        assert(json.containsKey(r'birthDate'),
            'Required key "ImportedPatient[birthDate]" is missing from JSON.');
        assert(json[r'birthDate'] != null,
            'Required key "ImportedPatient[birthDate]" has a null value in JSON.');
        return true;
      }());

      return ImportedPatient(
        id: mapValueOfType<String>(json, r'id'),
        givenName: mapValueOfType<String>(json, r'givenName')!,
        familyName: mapValueOfType<String>(json, r'familyName')!,
        birthDate: mapDateTime(json, r'birthDate', r'')!,
        phone: mapValueOfType<String>(json, r'phone'),
      );
    }
    return null;
  }

  static List<ImportedPatient> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ImportedPatient>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ImportedPatient.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ImportedPatient> mapFromJson(dynamic json) {
    final map = <String, ImportedPatient>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ImportedPatient.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ImportedPatient-objects as value to a dart map
  static Map<String, List<ImportedPatient>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ImportedPatient>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ImportedPatient.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'givenName',
    'familyName',
    'birthDate',
  };
}
