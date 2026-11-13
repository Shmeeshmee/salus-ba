//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class NoShowFlags {
  /// Returns a new [NoShowFlags] instance.
  NoShowFlags({
    this.flags = const [],
  });

  List<NoShowFlagsFlagsInner> flags;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is NoShowFlags && _deepEquality.equals(other.flags, flags);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (flags.hashCode);

  @override
  String toString() => 'NoShowFlags[flags=$flags]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'flags'] = this.flags;
    return json;
  }

  /// Returns a new [NoShowFlags] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static NoShowFlags? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'flags'),
            'Required key "NoShowFlags[flags]" is missing from JSON.');
        assert(json[r'flags'] != null,
            'Required key "NoShowFlags[flags]" has a null value in JSON.');
        return true;
      }());

      return NoShowFlags(
        flags: NoShowFlagsFlagsInner.listFromJson(json[r'flags']),
      );
    }
    return null;
  }

  static List<NoShowFlags> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <NoShowFlags>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = NoShowFlags.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, NoShowFlags> mapFromJson(dynamic json) {
    final map = <String, NoShowFlags>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = NoShowFlags.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of NoShowFlags-objects as value to a dart map
  static Map<String, List<NoShowFlags>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<NoShowFlags>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = NoShowFlags.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'flags',
  };
}
