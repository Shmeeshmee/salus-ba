//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DiagnosisGroupNameList {
  /// Returns a new [DiagnosisGroupNameList] instance.
  DiagnosisGroupNameList({
    this.groups = const [],
  });

  List<DiagnosisGroupName> groups;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DiagnosisGroupNameList &&
          _deepEquality.equals(other.groups, groups);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (groups.hashCode);

  @override
  String toString() => 'DiagnosisGroupNameList[groups=$groups]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'groups'] = this.groups;
    return json;
  }

  /// Returns a new [DiagnosisGroupNameList] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static DiagnosisGroupNameList? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'groups'),
            'Required key "DiagnosisGroupNameList[groups]" is missing from JSON.');
        assert(json[r'groups'] != null,
            'Required key "DiagnosisGroupNameList[groups]" has a null value in JSON.');
        return true;
      }());

      return DiagnosisGroupNameList(
        groups: DiagnosisGroupName.listFromJson(json[r'groups']),
      );
    }
    return null;
  }

  static List<DiagnosisGroupNameList> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DiagnosisGroupNameList>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DiagnosisGroupNameList.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, DiagnosisGroupNameList> mapFromJson(dynamic json) {
    final map = <String, DiagnosisGroupNameList>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = DiagnosisGroupNameList.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of DiagnosisGroupNameList-objects as value to a dart map
  static Map<String, List<DiagnosisGroupNameList>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<DiagnosisGroupNameList>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = DiagnosisGroupNameList.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'groups',
  };
}
