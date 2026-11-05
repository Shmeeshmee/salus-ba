//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Patient {
  /// Returns a new [Patient] instance.
  Patient({
    required this.id,
    required this.givenName,
    required this.familyName,
    required this.birthDate,
    this.phone,
    required this.versionId,
    required this.lastUpdated,
  });

  String id;

  String givenName;

  String familyName;

  DateTime birthDate;

  /// E.164; absent when none was given.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? phone;

  int versionId;

  DateTime lastUpdated;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Patient &&
          other.id == id &&
          other.givenName == givenName &&
          other.familyName == familyName &&
          other.birthDate == birthDate &&
          other.phone == phone &&
          other.versionId == versionId &&
          other.lastUpdated == lastUpdated;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (givenName.hashCode) +
      (familyName.hashCode) +
      (birthDate.hashCode) +
      (phone == null ? 0 : phone!.hashCode) +
      (versionId.hashCode) +
      (lastUpdated.hashCode);

  @override
  String toString() =>
      'Patient[id=$id, givenName=$givenName, familyName=$familyName, birthDate=$birthDate, phone=$phone, versionId=$versionId, lastUpdated=$lastUpdated]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'givenName'] = this.givenName;
    json[r'familyName'] = this.familyName;
    json[r'birthDate'] = _dateFormatter.format(this.birthDate);
    if (this.phone != null) {
      json[r'phone'] = this.phone;
    } else {
      json[r'phone'] = null;
    }
    json[r'versionId'] = this.versionId;
    json[r'lastUpdated'] = this.lastUpdated.toUtc().toIso8601String();
    return json;
  }

  /// Returns a new [Patient] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Patient? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "Patient[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "Patient[id]" has a null value in JSON.');
        assert(json.containsKey(r'givenName'),
            'Required key "Patient[givenName]" is missing from JSON.');
        assert(json[r'givenName'] != null,
            'Required key "Patient[givenName]" has a null value in JSON.');
        assert(json.containsKey(r'familyName'),
            'Required key "Patient[familyName]" is missing from JSON.');
        assert(json[r'familyName'] != null,
            'Required key "Patient[familyName]" has a null value in JSON.');
        assert(json.containsKey(r'birthDate'),
            'Required key "Patient[birthDate]" is missing from JSON.');
        assert(json[r'birthDate'] != null,
            'Required key "Patient[birthDate]" has a null value in JSON.');
        assert(json.containsKey(r'versionId'),
            'Required key "Patient[versionId]" is missing from JSON.');
        assert(json[r'versionId'] != null,
            'Required key "Patient[versionId]" has a null value in JSON.');
        assert(json.containsKey(r'lastUpdated'),
            'Required key "Patient[lastUpdated]" is missing from JSON.');
        assert(json[r'lastUpdated'] != null,
            'Required key "Patient[lastUpdated]" has a null value in JSON.');
        return true;
      }());

      return Patient(
        id: mapValueOfType<String>(json, r'id')!,
        givenName: mapValueOfType<String>(json, r'givenName')!,
        familyName: mapValueOfType<String>(json, r'familyName')!,
        birthDate: mapDateTime(json, r'birthDate', r'')!,
        phone: mapValueOfType<String>(json, r'phone'),
        versionId: mapValueOfType<int>(json, r'versionId')!,
        lastUpdated: mapDateTime(json, r'lastUpdated', r'')!,
      );
    }
    return null;
  }

  static List<Patient> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Patient>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Patient.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Patient> mapFromJson(dynamic json) {
    final map = <String, Patient>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Patient.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Patient-objects as value to a dart map
  static Map<String, List<Patient>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Patient>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Patient.listFromJson(
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
    'givenName',
    'familyName',
    'birthDate',
    'versionId',
    'lastUpdated',
  };
}
