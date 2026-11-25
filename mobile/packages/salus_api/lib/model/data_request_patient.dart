//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DataRequestPatient {
  /// Returns a new [DataRequestPatient] instance.
  DataRequestPatient({
    required this.givenName,
    required this.familyName,
    required this.birthDate,
    required this.registeredByClinic,
  });

  String givenName;

  String familyName;

  DateTime birthDate;

  /// The clinic registered the profile (no Salus account), so its admin hands the record over (GET /clinic/patients/{id}/record).
  bool registeredByClinic;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DataRequestPatient &&
          other.givenName == givenName &&
          other.familyName == familyName &&
          other.birthDate == birthDate &&
          other.registeredByClinic == registeredByClinic;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (givenName.hashCode) +
      (familyName.hashCode) +
      (birthDate.hashCode) +
      (registeredByClinic.hashCode);

  @override
  String toString() =>
      'DataRequestPatient[givenName=$givenName, familyName=$familyName, birthDate=$birthDate, registeredByClinic=$registeredByClinic]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'givenName'] = this.givenName;
    json[r'familyName'] = this.familyName;
    json[r'birthDate'] = _dateFormatter.format(this.birthDate);
    json[r'registeredByClinic'] = this.registeredByClinic;
    return json;
  }

  /// Returns a new [DataRequestPatient] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static DataRequestPatient? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'givenName'),
            'Required key "DataRequestPatient[givenName]" is missing from JSON.');
        assert(json[r'givenName'] != null,
            'Required key "DataRequestPatient[givenName]" has a null value in JSON.');
        assert(json.containsKey(r'familyName'),
            'Required key "DataRequestPatient[familyName]" is missing from JSON.');
        assert(json[r'familyName'] != null,
            'Required key "DataRequestPatient[familyName]" has a null value in JSON.');
        assert(json.containsKey(r'birthDate'),
            'Required key "DataRequestPatient[birthDate]" is missing from JSON.');
        assert(json[r'birthDate'] != null,
            'Required key "DataRequestPatient[birthDate]" has a null value in JSON.');
        assert(json.containsKey(r'registeredByClinic'),
            'Required key "DataRequestPatient[registeredByClinic]" is missing from JSON.');
        assert(json[r'registeredByClinic'] != null,
            'Required key "DataRequestPatient[registeredByClinic]" has a null value in JSON.');
        return true;
      }());

      return DataRequestPatient(
        givenName: mapValueOfType<String>(json, r'givenName')!,
        familyName: mapValueOfType<String>(json, r'familyName')!,
        birthDate: mapDateTime(json, r'birthDate', r'')!,
        registeredByClinic: mapValueOfType<bool>(json, r'registeredByClinic')!,
      );
    }
    return null;
  }

  static List<DataRequestPatient> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DataRequestPatient>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DataRequestPatient.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, DataRequestPatient> mapFromJson(dynamic json) {
    final map = <String, DataRequestPatient>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = DataRequestPatient.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of DataRequestPatient-objects as value to a dart map
  static Map<String, List<DataRequestPatient>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<DataRequestPatient>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = DataRequestPatient.listFromJson(
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
    'registeredByClinic',
  };
}
