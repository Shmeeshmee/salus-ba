//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class PlatformHeatmapCellsInner {
  /// Returns a new [PlatformHeatmapCellsInner] instance.
  PlatformHeatmapCellsInner({
    required this.weekday,
    required this.hour,
    required this.count,
  });

  /// Minimum value: 1
  /// Maximum value: 7
  int weekday;

  /// Minimum value: 0
  /// Maximum value: 23
  int hour;

  int? count;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is PlatformHeatmapCellsInner &&
          other.weekday == weekday &&
          other.hour == hour &&
          other.count == count;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (weekday.hashCode) +
      (hour.hashCode) +
      (count == null ? 0 : count!.hashCode);

  @override
  String toString() =>
      'PlatformHeatmapCellsInner[weekday=$weekday, hour=$hour, count=$count]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'weekday'] = this.weekday;
    json[r'hour'] = this.hour;
    if (this.count != null) {
      json[r'count'] = this.count;
    } else {
      json[r'count'] = null;
    }
    return json;
  }

  /// Returns a new [PlatformHeatmapCellsInner] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static PlatformHeatmapCellsInner? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'weekday'),
            'Required key "PlatformHeatmapCellsInner[weekday]" is missing from JSON.');
        assert(json[r'weekday'] != null,
            'Required key "PlatformHeatmapCellsInner[weekday]" has a null value in JSON.');
        assert(json.containsKey(r'hour'),
            'Required key "PlatformHeatmapCellsInner[hour]" is missing from JSON.');
        assert(json[r'hour'] != null,
            'Required key "PlatformHeatmapCellsInner[hour]" has a null value in JSON.');
        assert(json.containsKey(r'count'),
            'Required key "PlatformHeatmapCellsInner[count]" is missing from JSON.');
        return true;
      }());

      return PlatformHeatmapCellsInner(
        weekday: mapValueOfType<int>(json, r'weekday')!,
        hour: mapValueOfType<int>(json, r'hour')!,
        count: mapValueOfType<int>(json, r'count'),
      );
    }
    return null;
  }

  static List<PlatformHeatmapCellsInner> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <PlatformHeatmapCellsInner>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = PlatformHeatmapCellsInner.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, PlatformHeatmapCellsInner> mapFromJson(dynamic json) {
    final map = <String, PlatformHeatmapCellsInner>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = PlatformHeatmapCellsInner.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of PlatformHeatmapCellsInner-objects as value to a dart map
  static Map<String, List<PlatformHeatmapCellsInner>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<PlatformHeatmapCellsInner>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = PlatformHeatmapCellsInner.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'weekday',
    'hour',
    'count',
  };
}
