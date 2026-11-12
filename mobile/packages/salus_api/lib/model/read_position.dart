//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ReadPosition {
  /// Returns a new [ReadPosition] instance.
  ReadPosition({
    required this.seq,
  });

  /// Minimum value: 0
  int seq;

  @override
  bool operator ==(Object other) =>
      identical(this, other) || other is ReadPosition && other.seq == seq;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (seq.hashCode);

  @override
  String toString() => 'ReadPosition[seq=$seq]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'seq'] = this.seq;
    return json;
  }

  /// Returns a new [ReadPosition] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ReadPosition? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'seq'),
            'Required key "ReadPosition[seq]" is missing from JSON.');
        assert(json[r'seq'] != null,
            'Required key "ReadPosition[seq]" has a null value in JSON.');
        return true;
      }());

      return ReadPosition(
        seq: mapValueOfType<int>(json, r'seq')!,
      );
    }
    return null;
  }

  static List<ReadPosition> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ReadPosition>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ReadPosition.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ReadPosition> mapFromJson(dynamic json) {
    final map = <String, ReadPosition>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ReadPosition.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ReadPosition-objects as value to a dart map
  static Map<String, List<ReadPosition>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ReadPosition>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ReadPosition.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'seq',
  };
}
