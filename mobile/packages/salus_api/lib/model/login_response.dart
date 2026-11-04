//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class LoginResponse {
  /// Returns a new [LoginResponse] instance.
  LoginResponse({
    required this.next,
    this.challenge,
    this.token,
    this.expiresAt,
  });

  LoginResponseNextEnum next;

  /// For `verify` and `enrol`; valid 5 minutes.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? challenge;

  /// The session token, for `done` without `X-Salus-CSRF`; send it as `Authorization: Bearer <token>`.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? token;

  /// For `done`. Absolute; 12 hours for staff, 30 days for patients.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? expiresAt;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is LoginResponse &&
          other.next == next &&
          other.challenge == challenge &&
          other.token == token &&
          other.expiresAt == expiresAt;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (next.hashCode) +
      (challenge == null ? 0 : challenge!.hashCode) +
      (token == null ? 0 : token!.hashCode) +
      (expiresAt == null ? 0 : expiresAt!.hashCode);

  @override
  String toString() =>
      'LoginResponse[next=$next, challenge=$challenge, token=$token, expiresAt=$expiresAt]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'next'] = this.next;
    if (this.challenge != null) {
      json[r'challenge'] = this.challenge;
    } else {
      json[r'challenge'] = null;
    }
    if (this.token != null) {
      json[r'token'] = this.token;
    } else {
      json[r'token'] = null;
    }
    if (this.expiresAt != null) {
      json[r'expiresAt'] = this.expiresAt!.toUtc().toIso8601String();
    } else {
      json[r'expiresAt'] = null;
    }
    return json;
  }

  /// Returns a new [LoginResponse] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static LoginResponse? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'next'),
            'Required key "LoginResponse[next]" is missing from JSON.');
        assert(json[r'next'] != null,
            'Required key "LoginResponse[next]" has a null value in JSON.');
        return true;
      }());

      return LoginResponse(
        next: LoginResponseNextEnum.fromJson(json[r'next'])!,
        challenge: mapValueOfType<String>(json, r'challenge'),
        token: mapValueOfType<String>(json, r'token'),
        expiresAt: mapDateTime(json, r'expiresAt', r''),
      );
    }
    return null;
  }

  static List<LoginResponse> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <LoginResponse>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = LoginResponse.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, LoginResponse> mapFromJson(dynamic json) {
    final map = <String, LoginResponse>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = LoginResponse.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of LoginResponse-objects as value to a dart map
  static Map<String, List<LoginResponse>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<LoginResponse>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = LoginResponse.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'next',
  };
}

enum LoginResponseNextEnum {
  done._(r'done'),
  verify._(r'verify'),
  enrol._(r'enrol'),
  ;

  /// Instantiate a new enum with the provided value.
  const LoginResponseNextEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [LoginResponseNextEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static LoginResponseNextEnum? fromJson(dynamic value) =>
      LoginResponseNextEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [LoginResponseNextEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<LoginResponseNextEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <LoginResponseNextEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = LoginResponseNextEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [LoginResponseNextEnum] to String,
/// and [decode] dynamic data back to [LoginResponseNextEnum].
class LoginResponseNextEnumTypeTransformer {
  factory LoginResponseNextEnumTypeTransformer() =>
      _instance ??= const LoginResponseNextEnumTypeTransformer._();

  const LoginResponseNextEnumTypeTransformer._();

  String encode(LoginResponseNextEnum data) => data._value;

  /// Returns the instance of [LoginResponseNextEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  LoginResponseNextEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is LoginResponseNextEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'done':
          return LoginResponseNextEnum.done;
        case r'verify':
          return LoginResponseNextEnum.verify;
        case r'enrol':
          return LoginResponseNextEnum.enrol;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static LoginResponseNextEnumTypeTransformer? _instance;
}
