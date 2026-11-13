//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class QuietHours {
  /// Returns a new [QuietHours] instance.
  QuietHours({
    required this.start,
    required this.end,
  });

  String start;

  String end;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is QuietHours && other.start == start && other.end == end;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (start.hashCode) + (end.hashCode);

  @override
  String toString() => 'QuietHours[start=$start, end=$end]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'start'] = this.start;
    json[r'end'] = this.end;
    return json;
  }

  /// Returns a new [QuietHours] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static QuietHours? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'start'),
            'Required key "QuietHours[start]" is missing from JSON.');
        assert(json[r'start'] != null,
            'Required key "QuietHours[start]" has a null value in JSON.');
        assert(json.containsKey(r'end'),
            'Required key "QuietHours[end]" is missing from JSON.');
        assert(json[r'end'] != null,
            'Required key "QuietHours[end]" has a null value in JSON.');
        return true;
      }());

      return QuietHours(
        start: mapValueOfType<String>(json, r'start')!,
        end: mapValueOfType<String>(json, r'end')!,
      );
    }
    return null;
  }

  static List<QuietHours> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <QuietHours>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = QuietHours.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, QuietHours> mapFromJson(dynamic json) {
    final map = <String, QuietHours>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = QuietHours.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of QuietHours-objects as value to a dart map
  static Map<String, List<QuietHours>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<QuietHours>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = QuietHours.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'start',
    'end',
  };
}
