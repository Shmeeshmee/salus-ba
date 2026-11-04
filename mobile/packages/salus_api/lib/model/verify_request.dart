//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class VerifyRequest {
  /// Returns a new [VerifyRequest] instance.
  VerifyRequest({
    required this.challenge,
    required this.code,
  });

  String challenge;

  /// Six digits from the authenticator app.
  String code;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is VerifyRequest &&
          other.challenge == challenge &&
          other.code == code;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (challenge.hashCode) + (code.hashCode);

  @override
  String toString() => 'VerifyRequest[challenge=$challenge, code=$code]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'challenge'] = this.challenge;
    json[r'code'] = this.code;
    return json;
  }

  /// Returns a new [VerifyRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static VerifyRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'challenge'),
            'Required key "VerifyRequest[challenge]" is missing from JSON.');
        assert(json[r'challenge'] != null,
            'Required key "VerifyRequest[challenge]" has a null value in JSON.');
        assert(json.containsKey(r'code'),
            'Required key "VerifyRequest[code]" is missing from JSON.');
        assert(json[r'code'] != null,
            'Required key "VerifyRequest[code]" has a null value in JSON.');
        return true;
      }());

      return VerifyRequest(
        challenge: mapValueOfType<String>(json, r'challenge')!,
        code: mapValueOfType<String>(json, r'code')!,
      );
    }
    return null;
  }

  static List<VerifyRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <VerifyRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = VerifyRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, VerifyRequest> mapFromJson(dynamic json) {
    final map = <String, VerifyRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = VerifyRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of VerifyRequest-objects as value to a dart map
  static Map<String, List<VerifyRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<VerifyRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = VerifyRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'challenge',
    'code',
  };
}
