//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ClinicPatient {
  /// Returns a new [ClinicPatient] instance.
  ClinicPatient({
    required this.id,
    required this.givenName,
    required this.familyName,
    required this.birthDate,
    this.phone,
    required this.registeredByClinic,
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

  /// The clinic registered the profile; the patient has no Salus account.
  bool registeredByClinic;

  int versionId;

  DateTime lastUpdated;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ClinicPatient &&
          other.id == id &&
          other.givenName == givenName &&
          other.familyName == familyName &&
          other.birthDate == birthDate &&
          other.phone == phone &&
          other.registeredByClinic == registeredByClinic &&
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
      (registeredByClinic.hashCode) +
      (versionId.hashCode) +
      (lastUpdated.hashCode);

  @override
  String toString() =>
      'ClinicPatient[id=$id, givenName=$givenName, familyName=$familyName, birthDate=$birthDate, phone=$phone, registeredByClinic=$registeredByClinic, versionId=$versionId, lastUpdated=$lastUpdated]';

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
    json[r'registeredByClinic'] = this.registeredByClinic;
    json[r'versionId'] = this.versionId;
    json[r'lastUpdated'] = this.lastUpdated.toUtc().toIso8601String();
    return json;
  }

  /// Returns a new [ClinicPatient] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ClinicPatient? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "ClinicPatient[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "ClinicPatient[id]" has a null value in JSON.');
        assert(json.containsKey(r'givenName'),
            'Required key "ClinicPatient[givenName]" is missing from JSON.');
        assert(json[r'givenName'] != null,
            'Required key "ClinicPatient[givenName]" has a null value in JSON.');
        assert(json.containsKey(r'familyName'),
            'Required key "ClinicPatient[familyName]" is missing from JSON.');
        assert(json[r'familyName'] != null,
            'Required key "ClinicPatient[familyName]" has a null value in JSON.');
        assert(json.containsKey(r'birthDate'),
            'Required key "ClinicPatient[birthDate]" is missing from JSON.');
        assert(json[r'birthDate'] != null,
            'Required key "ClinicPatient[birthDate]" has a null value in JSON.');
        assert(json.containsKey(r'registeredByClinic'),
            'Required key "ClinicPatient[registeredByClinic]" is missing from JSON.');
        assert(json[r'registeredByClinic'] != null,
            'Required key "ClinicPatient[registeredByClinic]" has a null value in JSON.');
        assert(json.containsKey(r'versionId'),
            'Required key "ClinicPatient[versionId]" is missing from JSON.');
        assert(json[r'versionId'] != null,
            'Required key "ClinicPatient[versionId]" has a null value in JSON.');
        assert(json.containsKey(r'lastUpdated'),
            'Required key "ClinicPatient[lastUpdated]" is missing from JSON.');
        assert(json[r'lastUpdated'] != null,
            'Required key "ClinicPatient[lastUpdated]" has a null value in JSON.');
        return true;
      }());

      return ClinicPatient(
        id: mapValueOfType<String>(json, r'id')!,
        givenName: mapValueOfType<String>(json, r'givenName')!,
        familyName: mapValueOfType<String>(json, r'familyName')!,
        birthDate: mapDateTime(json, r'birthDate', r'')!,
        phone: mapValueOfType<String>(json, r'phone'),
        registeredByClinic: mapValueOfType<bool>(json, r'registeredByClinic')!,
        versionId: mapValueOfType<int>(json, r'versionId')!,
        lastUpdated: mapDateTime(json, r'lastUpdated', r'')!,
      );
    }
    return null;
  }

  static List<ClinicPatient> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ClinicPatient>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ClinicPatient.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ClinicPatient> mapFromJson(dynamic json) {
    final map = <String, ClinicPatient>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ClinicPatient.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ClinicPatient-objects as value to a dart map
  static Map<String, List<ClinicPatient>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ClinicPatient>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ClinicPatient.listFromJson(
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
    'registeredByClinic',
    'versionId',
    'lastUpdated',
  };
}
