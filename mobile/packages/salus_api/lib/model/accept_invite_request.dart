//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class AcceptInviteRequest {
  /// Returns a new [AcceptInviteRequest] instance.
  AcceptInviteRequest({
    required this.token,
    required this.password,
    this.language,
    this.detectedLanguage,
  });

  /// From the invite link's fragment.
  String token;

  /// Counted in characters, not bytes.
  String password;

  /// The language the user picked on the device before signing in; saved as their choice.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  Language? language;

  /// The language the client matched from the browser's or the device's languages; saved for e-mails while the user has no choice of their own.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  Language? detectedLanguage;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is AcceptInviteRequest &&
          other.token == token &&
          other.password == password &&
          other.language == language &&
          other.detectedLanguage == detectedLanguage;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (token.hashCode) +
      (password.hashCode) +
      (language == null ? 0 : language!.hashCode) +
      (detectedLanguage == null ? 0 : detectedLanguage!.hashCode);

  @override
  String toString() =>
      'AcceptInviteRequest[token=$token, password=$password, language=$language, detectedLanguage=$detectedLanguage]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'token'] = this.token;
    json[r'password'] = this.password;
    if (this.language != null) {
      json[r'language'] = this.language;
    } else {
      json[r'language'] = null;
    }
    if (this.detectedLanguage != null) {
      json[r'detectedLanguage'] = this.detectedLanguage;
    } else {
      json[r'detectedLanguage'] = null;
    }
    return json;
  }

  /// Returns a new [AcceptInviteRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static AcceptInviteRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'token'),
            'Required key "AcceptInviteRequest[token]" is missing from JSON.');
        assert(json[r'token'] != null,
            'Required key "AcceptInviteRequest[token]" has a null value in JSON.');
        assert(json.containsKey(r'password'),
            'Required key "AcceptInviteRequest[password]" is missing from JSON.');
        assert(json[r'password'] != null,
            'Required key "AcceptInviteRequest[password]" has a null value in JSON.');
        return true;
      }());

      return AcceptInviteRequest(
        token: mapValueOfType<String>(json, r'token')!,
        password: mapValueOfType<String>(json, r'password')!,
        language: Language.fromJson(json[r'language']),
        detectedLanguage: Language.fromJson(json[r'detectedLanguage']),
      );
    }
    return null;
  }

  static List<AcceptInviteRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AcceptInviteRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AcceptInviteRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, AcceptInviteRequest> mapFromJson(dynamic json) {
    final map = <String, AcceptInviteRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = AcceptInviteRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of AcceptInviteRequest-objects as value to a dart map
  static Map<String, List<AcceptInviteRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<AcceptInviteRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = AcceptInviteRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'token',
    'password',
  };
}
