//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Availability {
  /// Returns a new [Availability] instance.
  Availability({
    this.id,
    required this.practitionerRoleId,
    this.slotMinutes,
    this.availability = const [],
    required this.versionId,
  });

  /// The schedule's id (FHIR Schedule); absent before the first edit.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? id;

  String practitionerRoleId;

  /// Absent before the first edit.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  int? slotMinutes;

  /// Sorted by day and start.
  List<AvailabilityRule> availability;

  /// 0 before the first edit.
  int versionId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Availability &&
          other.id == id &&
          other.practitionerRoleId == practitionerRoleId &&
          other.slotMinutes == slotMinutes &&
          _deepEquality.equals(other.availability, availability) &&
          other.versionId == versionId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id == null ? 0 : id!.hashCode) +
      (practitionerRoleId.hashCode) +
      (slotMinutes == null ? 0 : slotMinutes!.hashCode) +
      (availability.hashCode) +
      (versionId.hashCode);

  @override
  String toString() =>
      'Availability[id=$id, practitionerRoleId=$practitionerRoleId, slotMinutes=$slotMinutes, availability=$availability, versionId=$versionId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    if (this.id != null) {
      json[r'id'] = this.id;
    } else {
      json[r'id'] = null;
    }
    json[r'practitionerRoleId'] = this.practitionerRoleId;
    if (this.slotMinutes != null) {
      json[r'slotMinutes'] = this.slotMinutes;
    } else {
      json[r'slotMinutes'] = null;
    }
    json[r'availability'] = this.availability;
    json[r'versionId'] = this.versionId;
    return json;
  }

  /// Returns a new [Availability] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Availability? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'practitionerRoleId'),
            'Required key "Availability[practitionerRoleId]" is missing from JSON.');
        assert(json[r'practitionerRoleId'] != null,
            'Required key "Availability[practitionerRoleId]" has a null value in JSON.');
        assert(json.containsKey(r'availability'),
            'Required key "Availability[availability]" is missing from JSON.');
        assert(json[r'availability'] != null,
            'Required key "Availability[availability]" has a null value in JSON.');
        assert(json.containsKey(r'versionId'),
            'Required key "Availability[versionId]" is missing from JSON.');
        assert(json[r'versionId'] != null,
            'Required key "Availability[versionId]" has a null value in JSON.');
        return true;
      }());

      return Availability(
        id: mapValueOfType<String>(json, r'id'),
        practitionerRoleId:
            mapValueOfType<String>(json, r'practitionerRoleId')!,
        slotMinutes: mapValueOfType<int>(json, r'slotMinutes'),
        availability: AvailabilityRule.listFromJson(json[r'availability']),
        versionId: mapValueOfType<int>(json, r'versionId')!,
      );
    }
    return null;
  }

  static List<Availability> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Availability>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Availability.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Availability> mapFromJson(dynamic json) {
    final map = <String, Availability>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Availability.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Availability-objects as value to a dart map
  static Map<String, List<Availability>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Availability>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Availability.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'practitionerRoleId',
    'availability',
    'versionId',
  };
}
