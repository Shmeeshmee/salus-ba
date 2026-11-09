//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class KeyedError {
  /// Returns a new [KeyedError] instance.
  KeyedError({
    required this.error,
    this.keys = const [],
  });

  String error;

  List<String> keys;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is KeyedError &&
          other.error == error &&
          _deepEquality.equals(other.keys, keys);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (error.hashCode) + (keys.hashCode);

  @override
  String toString() => 'KeyedError[error=$error, keys=$keys]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'error'] = this.error;
    json[r'keys'] = this.keys;
    return json;
  }

  /// Returns a new [KeyedError] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static KeyedError? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'error'),
            'Required key "KeyedError[error]" is missing from JSON.');
        assert(json[r'error'] != null,
            'Required key "KeyedError[error]" has a null value in JSON.');
        assert(json.containsKey(r'keys'),
            'Required key "KeyedError[keys]" is missing from JSON.');
        assert(json[r'keys'] != null,
            'Required key "KeyedError[keys]" has a null value in JSON.');
        return true;
      }());

      return KeyedError(
        error: mapValueOfType<String>(json, r'error')!,
        keys: json[r'keys'] is Iterable
            ? (json[r'keys'] as Iterable).cast<String>().toList(growable: false)
            : const [],
      );
    }
    return null;
  }

  static List<KeyedError> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <KeyedError>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = KeyedError.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, KeyedError> mapFromJson(dynamic json) {
    final map = <String, KeyedError>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = KeyedError.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of KeyedError-objects as value to a dart map
  static Map<String, List<KeyedError>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<KeyedError>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = KeyedError.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'error',
    'keys',
  };
}
