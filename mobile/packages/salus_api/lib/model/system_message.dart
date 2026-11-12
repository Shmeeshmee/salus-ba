//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SystemMessage {
  /// Returns a new [SystemMessage] instance.
  SystemMessage({
    required this.code,
    this.args = const {},
  });

  String code;

  Map<String, String> args;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SystemMessage &&
          other.code == code &&
          _deepEquality.equals(other.args, args);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (code.hashCode) + (args.hashCode);

  @override
  String toString() => 'SystemMessage[code=$code, args=$args]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'code'] = this.code;
    json[r'args'] = this.args;
    return json;
  }

  /// Returns a new [SystemMessage] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SystemMessage? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'code'),
            'Required key "SystemMessage[code]" is missing from JSON.');
        assert(json[r'code'] != null,
            'Required key "SystemMessage[code]" has a null value in JSON.');
        return true;
      }());

      return SystemMessage(
        code: mapValueOfType<String>(json, r'code')!,
        args: mapCastOfType<String, String>(json, r'args') ?? const {},
      );
    }
    return null;
  }

  static List<SystemMessage> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SystemMessage>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SystemMessage.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SystemMessage> mapFromJson(dynamic json) {
    final map = <String, SystemMessage>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SystemMessage.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SystemMessage-objects as value to a dart map
  static Map<String, List<SystemMessage>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SystemMessage>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SystemMessage.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'code',
  };
}
