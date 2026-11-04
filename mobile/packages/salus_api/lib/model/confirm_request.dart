//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ConfirmRequest {
  /// Returns a new [ConfirmRequest] instance.
  ConfirmRequest({
    this.challenge,
    required this.code,
  });

  /// The login challenge; omitted when enrolling from a session.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? challenge;

  /// Six digits from the authenticator app.
  String code;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ConfirmRequest &&
          other.challenge == challenge &&
          other.code == code;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (challenge == null ? 0 : challenge!.hashCode) + (code.hashCode);

  @override
  String toString() => 'ConfirmRequest[challenge=$challenge, code=$code]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    if (this.challenge != null) {
      json[r'challenge'] = this.challenge;
    } else {
      json[r'challenge'] = null;
    }
    json[r'code'] = this.code;
    return json;
  }

  /// Returns a new [ConfirmRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ConfirmRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'code'),
            'Required key "ConfirmRequest[code]" is missing from JSON.');
        assert(json[r'code'] != null,
            'Required key "ConfirmRequest[code]" has a null value in JSON.');
        return true;
      }());

      return ConfirmRequest(
        challenge: mapValueOfType<String>(json, r'challenge'),
        code: mapValueOfType<String>(json, r'code')!,
      );
    }
    return null;
  }

  static List<ConfirmRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ConfirmRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ConfirmRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ConfirmRequest> mapFromJson(dynamic json) {
    final map = <String, ConfirmRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ConfirmRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ConfirmRequest-objects as value to a dart map
  static Map<String, List<ConfirmRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ConfirmRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ConfirmRequest.listFromJson(
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
