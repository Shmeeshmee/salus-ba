//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class RoleSpecialty {
  /// Returns a new [RoleSpecialty] instance.
  RoleSpecialty({
    required this.id,
    this.specialty,
    required this.versionId,
  });

  String id;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  Specialty? specialty;

  int versionId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is RoleSpecialty &&
          other.id == id &&
          other.specialty == specialty &&
          other.versionId == versionId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (specialty == null ? 0 : specialty!.hashCode) +
      (versionId.hashCode);

  @override
  String toString() =>
      'RoleSpecialty[id=$id, specialty=$specialty, versionId=$versionId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    if (this.specialty != null) {
      json[r'specialty'] = this.specialty;
    } else {
      json[r'specialty'] = null;
    }
    json[r'versionId'] = this.versionId;
    return json;
  }

  /// Returns a new [RoleSpecialty] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static RoleSpecialty? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "RoleSpecialty[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "RoleSpecialty[id]" has a null value in JSON.');
        assert(json.containsKey(r'versionId'),
            'Required key "RoleSpecialty[versionId]" is missing from JSON.');
        assert(json[r'versionId'] != null,
            'Required key "RoleSpecialty[versionId]" has a null value in JSON.');
        return true;
      }());

      return RoleSpecialty(
        id: mapValueOfType<String>(json, r'id')!,
        specialty: Specialty.fromJson(json[r'specialty']),
        versionId: mapValueOfType<int>(json, r'versionId')!,
      );
    }
    return null;
  }

  static List<RoleSpecialty> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <RoleSpecialty>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = RoleSpecialty.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, RoleSpecialty> mapFromJson(dynamic json) {
    final map = <String, RoleSpecialty>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = RoleSpecialty.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of RoleSpecialty-objects as value to a dart map
  static Map<String, List<RoleSpecialty>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<RoleSpecialty>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = RoleSpecialty.listFromJson(
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
    'versionId',
  };
}
