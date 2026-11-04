//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Acceptance {
  /// Returns a new [Acceptance] instance.
  Acceptance({
    required this.challenge,
    required this.secret,
    required this.uri,
  });

  /// For `/auth/mfa/confirm`; valid 5 minutes.
  String challenge;

  /// Base32, for manual entry.
  String secret;

  /// The otpauth:// URI, for a QR code.
  String uri;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Acceptance &&
          other.challenge == challenge &&
          other.secret == secret &&
          other.uri == uri;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (challenge.hashCode) + (secret.hashCode) + (uri.hashCode);

  @override
  String toString() =>
      'Acceptance[challenge=$challenge, secret=$secret, uri=$uri]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'challenge'] = this.challenge;
    json[r'secret'] = this.secret;
    json[r'uri'] = this.uri;
    return json;
  }

  /// Returns a new [Acceptance] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Acceptance? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'challenge'),
            'Required key "Acceptance[challenge]" is missing from JSON.');
        assert(json[r'challenge'] != null,
            'Required key "Acceptance[challenge]" has a null value in JSON.');
        assert(json.containsKey(r'secret'),
            'Required key "Acceptance[secret]" is missing from JSON.');
        assert(json[r'secret'] != null,
            'Required key "Acceptance[secret]" has a null value in JSON.');
        assert(json.containsKey(r'uri'),
            'Required key "Acceptance[uri]" is missing from JSON.');
        assert(json[r'uri'] != null,
            'Required key "Acceptance[uri]" has a null value in JSON.');
        return true;
      }());

      return Acceptance(
        challenge: mapValueOfType<String>(json, r'challenge')!,
        secret: mapValueOfType<String>(json, r'secret')!,
        uri: mapValueOfType<String>(json, r'uri')!,
      );
    }
    return null;
  }

  static List<Acceptance> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Acceptance>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Acceptance.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Acceptance> mapFromJson(dynamic json) {
    final map = <String, Acceptance>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Acceptance.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Acceptance-objects as value to a dart map
  static Map<String, List<Acceptance>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Acceptance>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Acceptance.listFromJson(
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
    'secret',
    'uri',
  };
}
