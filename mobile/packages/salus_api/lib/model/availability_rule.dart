//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class AvailabilityRule {
  /// Returns a new [AvailabilityRule] instance.
  AvailabilityRule({
    required this.day,
    required this.start,
    required this.end,
  });

  /// ISO weekday, 1 Monday to 7 Sunday.
  ///
  /// Minimum value: 1
  /// Maximum value: 7
  int day;

  String start;

  /// After start.
  String end;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is AvailabilityRule &&
          other.day == day &&
          other.start == start &&
          other.end == end;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (day.hashCode) + (start.hashCode) + (end.hashCode);

  @override
  String toString() => 'AvailabilityRule[day=$day, start=$start, end=$end]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'day'] = this.day;
    json[r'start'] = this.start;
    json[r'end'] = this.end;
    return json;
  }

  /// Returns a new [AvailabilityRule] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static AvailabilityRule? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'day'),
            'Required key "AvailabilityRule[day]" is missing from JSON.');
        assert(json[r'day'] != null,
            'Required key "AvailabilityRule[day]" has a null value in JSON.');
        assert(json.containsKey(r'start'),
            'Required key "AvailabilityRule[start]" is missing from JSON.');
        assert(json[r'start'] != null,
            'Required key "AvailabilityRule[start]" has a null value in JSON.');
        assert(json.containsKey(r'end'),
            'Required key "AvailabilityRule[end]" is missing from JSON.');
        assert(json[r'end'] != null,
            'Required key "AvailabilityRule[end]" has a null value in JSON.');
        return true;
      }());

      return AvailabilityRule(
        day: mapValueOfType<int>(json, r'day')!,
        start: mapValueOfType<String>(json, r'start')!,
        end: mapValueOfType<String>(json, r'end')!,
      );
    }
    return null;
  }

  static List<AvailabilityRule> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AvailabilityRule>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AvailabilityRule.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, AvailabilityRule> mapFromJson(dynamic json) {
    final map = <String, AvailabilityRule>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = AvailabilityRule.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of AvailabilityRule-objects as value to a dart map
  static Map<String, List<AvailabilityRule>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<AvailabilityRule>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = AvailabilityRule.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'day',
    'start',
    'end',
  };
}
