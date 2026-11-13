//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class WaitlistPriority {
  /// Returns a new [WaitlistPriority] instance.
  WaitlistPriority({
    required this.priority,
  });

  /// Minimum value: 0
  /// Maximum value: 100
  int priority;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is WaitlistPriority && other.priority == priority;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (priority.hashCode);

  @override
  String toString() => 'WaitlistPriority[priority=$priority]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'priority'] = this.priority;
    return json;
  }

  /// Returns a new [WaitlistPriority] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static WaitlistPriority? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'priority'),
            'Required key "WaitlistPriority[priority]" is missing from JSON.');
        assert(json[r'priority'] != null,
            'Required key "WaitlistPriority[priority]" has a null value in JSON.');
        return true;
      }());

      return WaitlistPriority(
        priority: mapValueOfType<int>(json, r'priority')!,
      );
    }
    return null;
  }

  static List<WaitlistPriority> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <WaitlistPriority>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = WaitlistPriority.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, WaitlistPriority> mapFromJson(dynamic json) {
    final map = <String, WaitlistPriority>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = WaitlistPriority.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of WaitlistPriority-objects as value to a dart map
  static Map<String, List<WaitlistPriority>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<WaitlistPriority>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = WaitlistPriority.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'priority',
  };
}
