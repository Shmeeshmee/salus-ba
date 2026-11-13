//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class WaitlistEntry {
  /// Returns a new [WaitlistEntry] instance.
  WaitlistEntry({
    required this.id,
    required this.organizationId,
    this.practitionerRoleId,
    this.specialty,
    required this.earliest,
    required this.latest,
    this.times,
    required this.status,
    required this.createdAt,
    this.offer,
  });

  String id;

  String organizationId;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? practitionerRoleId;

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

  WaitlistStatus status;

  DateTime createdAt;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  WaitlistOffer? offer;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is WaitlistEntry &&
          other.id == id &&
          other.organizationId == organizationId &&
          other.practitionerRoleId == practitionerRoleId &&
          other.specialty == specialty &&
          other.earliest == earliest &&
          other.latest == latest &&
          other.times == times &&
          other.status == status &&
          other.createdAt == createdAt &&
          other.offer == offer;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (organizationId.hashCode) +
      (practitionerRoleId == null ? 0 : practitionerRoleId!.hashCode) +
      (specialty == null ? 0 : specialty!.hashCode) +
      (earliest.hashCode) +
      (latest.hashCode) +
      (times == null ? 0 : times!.hashCode) +
      (status.hashCode) +
      (createdAt.hashCode) +
      (offer == null ? 0 : offer!.hashCode);

  @override
  String toString() =>
      'WaitlistEntry[id=$id, organizationId=$organizationId, practitionerRoleId=$practitionerRoleId, specialty=$specialty, earliest=$earliest, latest=$latest, times=$times, status=$status, createdAt=$createdAt, offer=$offer]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
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
    json[r'status'] = this.status;
    json[r'created_at'] = this.createdAt.toUtc().toIso8601String();
    if (this.offer != null) {
      json[r'offer'] = this.offer;
    } else {
      json[r'offer'] = null;
    }
    return json;
  }

  /// Returns a new [WaitlistEntry] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static WaitlistEntry? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "WaitlistEntry[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "WaitlistEntry[id]" has a null value in JSON.');
        assert(json.containsKey(r'organization_id'),
            'Required key "WaitlistEntry[organization_id]" is missing from JSON.');
        assert(json[r'organization_id'] != null,
            'Required key "WaitlistEntry[organization_id]" has a null value in JSON.');
        assert(json.containsKey(r'earliest'),
            'Required key "WaitlistEntry[earliest]" is missing from JSON.');
        assert(json[r'earliest'] != null,
            'Required key "WaitlistEntry[earliest]" has a null value in JSON.');
        assert(json.containsKey(r'latest'),
            'Required key "WaitlistEntry[latest]" is missing from JSON.');
        assert(json[r'latest'] != null,
            'Required key "WaitlistEntry[latest]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "WaitlistEntry[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "WaitlistEntry[status]" has a null value in JSON.');
        assert(json.containsKey(r'created_at'),
            'Required key "WaitlistEntry[created_at]" is missing from JSON.');
        assert(json[r'created_at'] != null,
            'Required key "WaitlistEntry[created_at]" has a null value in JSON.');
        return true;
      }());

      return WaitlistEntry(
        id: mapValueOfType<String>(json, r'id')!,
        organizationId: mapValueOfType<String>(json, r'organization_id')!,
        practitionerRoleId:
            mapValueOfType<String>(json, r'practitioner_role_id'),
        specialty: mapValueOfType<String>(json, r'specialty'),
        earliest: mapDateTime(json, r'earliest', r'')!,
        latest: mapDateTime(json, r'latest', r'')!,
        times: WaitlistTimes.fromJson(json[r'times']),
        status: WaitlistStatus.fromJson(json[r'status'])!,
        createdAt: mapDateTime(json, r'created_at', r'')!,
        offer: WaitlistOffer.fromJson(json[r'offer']),
      );
    }
    return null;
  }

  static List<WaitlistEntry> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <WaitlistEntry>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = WaitlistEntry.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, WaitlistEntry> mapFromJson(dynamic json) {
    final map = <String, WaitlistEntry>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = WaitlistEntry.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of WaitlistEntry-objects as value to a dart map
  static Map<String, List<WaitlistEntry>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<WaitlistEntry>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = WaitlistEntry.listFromJson(
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
    'organization_id',
    'earliest',
    'latest',
    'status',
    'created_at',
  };
}
