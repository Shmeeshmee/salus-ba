//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class NewWaitlistEntry {
  /// Returns a new [NewWaitlistEntry] instance.
  NewWaitlistEntry({
    required this.organizationId,
    this.practitionerRoleId,
    this.specialty,
    required this.earliest,
    required this.latest,
    this.times,
  });

  String organizationId;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? practitionerRoleId;

  /// A specialty code of GET /specialties.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? specialty;

  DateTime earliest;

  DateTime latest;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  WaitlistTimes? times;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is NewWaitlistEntry &&
          other.organizationId == organizationId &&
          other.practitionerRoleId == practitionerRoleId &&
          other.specialty == specialty &&
          other.earliest == earliest &&
          other.latest == latest &&
          other.times == times;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (organizationId.hashCode) +
      (practitionerRoleId == null ? 0 : practitionerRoleId!.hashCode) +
      (specialty == null ? 0 : specialty!.hashCode) +
      (earliest.hashCode) +
      (latest.hashCode) +
      (times == null ? 0 : times!.hashCode);

  @override
  String toString() =>
      'NewWaitlistEntry[organizationId=$organizationId, practitionerRoleId=$practitionerRoleId, specialty=$specialty, earliest=$earliest, latest=$latest, times=$times]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'organization_id'] = this.organizationId;
    if (this.practitionerRoleId != null) {
      json[r'practitioner_role_id'] = this.practitionerRoleId;
    } else {
      json[r'practitioner_role_id'] = null;
    }
    if (this.specialty != null) {
      json[r'specialty'] = this.specialty;
    } else {
      json[r'specialty'] = null;
    }
    json[r'earliest'] = _dateFormatter.format(this.earliest);
    json[r'latest'] = _dateFormatter.format(this.latest);
    if (this.times != null) {
      json[r'times'] = this.times;
    } else {
      json[r'times'] = null;
    }
    return json;
  }

  /// Returns a new [NewWaitlistEntry] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static NewWaitlistEntry? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'organization_id'),
            'Required key "NewWaitlistEntry[organization_id]" is missing from JSON.');
        assert(json[r'organization_id'] != null,
            'Required key "NewWaitlistEntry[organization_id]" has a null value in JSON.');
        assert(json.containsKey(r'earliest'),
            'Required key "NewWaitlistEntry[earliest]" is missing from JSON.');
        assert(json[r'earliest'] != null,
            'Required key "NewWaitlistEntry[earliest]" has a null value in JSON.');
        assert(json.containsKey(r'latest'),
            'Required key "NewWaitlistEntry[latest]" is missing from JSON.');
        assert(json[r'latest'] != null,
            'Required key "NewWaitlistEntry[latest]" has a null value in JSON.');
        return true;
      }());

      return NewWaitlistEntry(
        organizationId: mapValueOfType<String>(json, r'organization_id')!,
        practitionerRoleId:
            mapValueOfType<String>(json, r'practitioner_role_id'),
        specialty: mapValueOfType<String>(json, r'specialty'),
        earliest: mapDateTime(json, r'earliest', r'')!,
        latest: mapDateTime(json, r'latest', r'')!,
        times: WaitlistTimes.fromJson(json[r'times']),
      );
    }
    return null;
  }

  static List<NewWaitlistEntry> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <NewWaitlistEntry>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = NewWaitlistEntry.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, NewWaitlistEntry> mapFromJson(dynamic json) {
    final map = <String, NewWaitlistEntry>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = NewWaitlistEntry.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of NewWaitlistEntry-objects as value to a dart map
  static Map<String, List<NewWaitlistEntry>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<NewWaitlistEntry>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = NewWaitlistEntry.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'organization_id',
    'earliest',
    'latest',
  };
}
