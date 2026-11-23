//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SignupRequest {
  /// Returns a new [SignupRequest] instance.
  SignupRequest({
    required this.email,
    required this.clinicName,
    required this.language,
  });

  /// The clinic admin's; becomes their login.
  String email;

  String clinicName;

  Language language;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SignupRequest &&
          other.email == email &&
          other.clinicName == clinicName &&
          other.language == language;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (email.hashCode) + (clinicName.hashCode) + (language.hashCode);

  @override
  String toString() =>
      'SignupRequest[email=$email, clinicName=$clinicName, language=$language]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'email'] = this.email;
    json[r'clinicName'] = this.clinicName;
    json[r'language'] = this.language;
    return json;
  }

  /// Returns a new [SignupRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SignupRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'email'),
            'Required key "SignupRequest[email]" is missing from JSON.');
        assert(json[r'email'] != null,
            'Required key "SignupRequest[email]" has a null value in JSON.');
        assert(json.containsKey(r'clinicName'),
            'Required key "SignupRequest[clinicName]" is missing from JSON.');
        assert(json[r'clinicName'] != null,
            'Required key "SignupRequest[clinicName]" has a null value in JSON.');
        assert(json.containsKey(r'language'),
            'Required key "SignupRequest[language]" is missing from JSON.');
        assert(json[r'language'] != null,
            'Required key "SignupRequest[language]" has a null value in JSON.');
        return true;
      }());

      return SignupRequest(
        email: mapValueOfType<String>(json, r'email')!,
        clinicName: mapValueOfType<String>(json, r'clinicName')!,
        language: Language.fromJson(json[r'language'])!,
      );
    }
    return null;
  }

  static List<SignupRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SignupRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SignupRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SignupRequest> mapFromJson(dynamic json) {
    final map = <String, SignupRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SignupRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SignupRequest-objects as value to a dart map
  static Map<String, List<SignupRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SignupRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SignupRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'email',
    'clinicName',
    'language',
  };
}
