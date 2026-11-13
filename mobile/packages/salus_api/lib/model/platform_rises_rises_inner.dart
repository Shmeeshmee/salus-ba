//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class PlatformRisesRisesInner {
  /// Returns a new [PlatformRisesRisesInner] instance.
  PlatformRisesRisesInner({
    required this.week,
    required this.region,
    required this.group,
  });

  String week;

  String region;

  String group;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is PlatformRisesRisesInner &&
          other.week == week &&
          other.region == region &&
          other.group == group;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (week.hashCode) + (region.hashCode) + (group.hashCode);

  @override
  String toString() =>
      'PlatformRisesRisesInner[week=$week, region=$region, group=$group]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'week'] = this.week;
    json[r'region'] = this.region;
    json[r'group'] = this.group;
    return json;
  }

  /// Returns a new [PlatformRisesRisesInner] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static PlatformRisesRisesInner? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'week'),
            'Required key "PlatformRisesRisesInner[week]" is missing from JSON.');
        assert(json[r'week'] != null,
            'Required key "PlatformRisesRisesInner[week]" has a null value in JSON.');
        assert(json.containsKey(r'region'),
            'Required key "PlatformRisesRisesInner[region]" is missing from JSON.');
        assert(json[r'region'] != null,
            'Required key "PlatformRisesRisesInner[region]" has a null value in JSON.');
        assert(json.containsKey(r'group'),
            'Required key "PlatformRisesRisesInner[group]" is missing from JSON.');
        assert(json[r'group'] != null,
            'Required key "PlatformRisesRisesInner[group]" has a null value in JSON.');
        return true;
      }());

      return PlatformRisesRisesInner(
        week: mapValueOfType<String>(json, r'week')!,
        region: mapValueOfType<String>(json, r'region')!,
        group: mapValueOfType<String>(json, r'group')!,
      );
    }
    return null;
  }

  static List<PlatformRisesRisesInner> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <PlatformRisesRisesInner>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = PlatformRisesRisesInner.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, PlatformRisesRisesInner> mapFromJson(dynamic json) {
    final map = <String, PlatformRisesRisesInner>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = PlatformRisesRisesInner.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of PlatformRisesRisesInner-objects as value to a dart map
  static Map<String, List<PlatformRisesRisesInner>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<PlatformRisesRisesInner>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = PlatformRisesRisesInner.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'week',
    'region',
    'group',
  };
}
