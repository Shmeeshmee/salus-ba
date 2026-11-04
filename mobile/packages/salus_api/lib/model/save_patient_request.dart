//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SavePatientRequest {
  /// Returns a new [SavePatientRequest] instance.
  SavePatientRequest({
    required this.givenName,
    required this.familyName,
    required this.birthDate,
    this.phone,
    this.versionId,
  });

  /// 1 to 100 characters after trimming.
  String givenName;

  /// 1 to 100 characters after trimming.
  String familyName;

  /// Before today (UTC), not before 1900-01-01.
  DateTime birthDate;

  /// Optional, E.164 (+ and 8 to 15 digits, no spaces); absent for none.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? phone;

  /// The version being updated; absent to create the profile.
  ///
  /// Minimum value: 1
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  int? versionId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SavePatientRequest &&
          other.givenName == givenName &&
          other.familyName == familyName &&
          other.birthDate == birthDate &&
          other.phone == phone &&
          other.versionId == versionId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (givenName.hashCode) +
      (familyName.hashCode) +
      (birthDate.hashCode) +
      (phone == null ? 0 : phone!.hashCode) +
      (versionId == null ? 0 : versionId!.hashCode);

  @override
  String toString() =>
      'SavePatientRequest[givenName=$givenName, familyName=$familyName, birthDate=$birthDate, phone=$phone, versionId=$versionId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'givenName'] = this.givenName;
    json[r'familyName'] = this.familyName;
    json[r'birthDate'] = _dateFormatter.format(this.birthDate);
    if (this.phone != null) {
      json[r'phone'] = this.phone;
    } else {
      json[r'phone'] = null;
    }
    if (this.versionId != null) {
      json[r'versionId'] = this.versionId;
    } else {
      json[r'versionId'] = null;
    }
    return json;
  }

  /// Returns a new [SavePatientRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SavePatientRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'givenName'),
            'Required key "SavePatientRequest[givenName]" is missing from JSON.');
        assert(json[r'givenName'] != null,
            'Required key "SavePatientRequest[givenName]" has a null value in JSON.');
        assert(json.containsKey(r'familyName'),
            'Required key "SavePatientRequest[familyName]" is missing from JSON.');
        assert(json[r'familyName'] != null,
            'Required key "SavePatientRequest[familyName]" has a null value in JSON.');
        assert(json.containsKey(r'birthDate'),
            'Required key "SavePatientRequest[birthDate]" is missing from JSON.');
        assert(json[r'birthDate'] != null,
            'Required key "SavePatientRequest[birthDate]" has a null value in JSON.');
        return true;
      }());

      return SavePatientRequest(
        givenName: mapValueOfType<String>(json, r'givenName')!,
        familyName: mapValueOfType<String>(json, r'familyName')!,
        birthDate: mapDateTime(json, r'birthDate', r'')!,
        phone: mapValueOfType<String>(json, r'phone'),
        versionId: mapValueOfType<int>(json, r'versionId'),
      );
    }
    return null;
  }

  static List<SavePatientRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SavePatientRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SavePatientRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SavePatientRequest> mapFromJson(dynamic json) {
    final map = <String, SavePatientRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SavePatientRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SavePatientRequest-objects as value to a dart map
  static Map<String, List<SavePatientRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SavePatientRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SavePatientRequest.listFromJson(
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
