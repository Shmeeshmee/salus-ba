//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Error {
  /// Returns a new [Error] instance.
  Error({
    required this.error,
  });

  ErrorErrorEnum error;

  @override
  bool operator ==(Object other) =>
      identical(this, other) || other is Error && other.error == error;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (error.hashCode);

  @override
  String toString() => 'Error[error=$error]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'error'] = this.error;
    return json;
  }

  /// Returns a new [Error] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Error? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'error'),
            'Required key "Error[error]" is missing from JSON.');
        assert(json[r'error'] != null,
            'Required key "Error[error]" has a null value in JSON.');
        return true;
      }());

      return Error(
        error: ErrorErrorEnum.fromJson(json[r'error'])!,
      );
    }
    return null;
  }

  static List<Error> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Error>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Error.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Error> mapFromJson(dynamic json) {
    final map = <String, Error>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Error.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Error-objects as value to a dart map
  static Map<String, List<Error>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Error>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Error.listFromJson(
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
  };
}

enum ErrorErrorEnum {
  invalidRequest._(r'invalid_request'),
  invalidEmail._(r'invalid_email'),
  invalidPassword._(r'invalid_password'),
  emailTaken._(r'email_taken'),
  unauthenticated._(r'unauthenticated'),
  wrongCredentials._(r'wrong_credentials'),
  rateLimited._(r'rate_limited'),
  invalidChallenge._(r'invalid_challenge'),
  wrongCode._(r'wrong_code'),
  alreadyEnrolled._(r'already_enrolled'),
  notEnrolled._(r'not_enrolled'),
  csrf._(r'csrf'),
  forbidden._(r'forbidden'),
  invalidRole._(r'invalid_role'),
  invalidName._(r'invalid_name'),
  invalidInvite._(r'invalid_invite'),
  invalidBirthDate._(r'invalid_birth_date'),
  invalidPhone._(r'invalid_phone'),
  notFound._(r'not_found'),
  versionConflict._(r'version_conflict'),
  consentActive._(r'consent_active'),
  consentInactive._(r'consent_inactive'),
  profileRequired._(r'profile_required'),
  internalError._(r'internal error'),
  timeout._(r'timeout'),
  ;

  /// Instantiate a new enum with the provided value.
  const ErrorErrorEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ErrorErrorEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ErrorErrorEnum? fromJson(dynamic value) =>
      ErrorErrorEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ErrorErrorEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ErrorErrorEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ErrorErrorEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ErrorErrorEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ErrorErrorEnum] to String,
/// and [decode] dynamic data back to [ErrorErrorEnum].
class ErrorErrorEnumTypeTransformer {
  factory ErrorErrorEnumTypeTransformer() =>
      _instance ??= const ErrorErrorEnumTypeTransformer._();

  const ErrorErrorEnumTypeTransformer._();

  String encode(ErrorErrorEnum data) => data._value;

  /// Returns the instance of [ErrorErrorEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ErrorErrorEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ErrorErrorEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'invalid_request':
          return ErrorErrorEnum.invalidRequest;
        case r'invalid_email':
          return ErrorErrorEnum.invalidEmail;
        case r'invalid_password':
          return ErrorErrorEnum.invalidPassword;
        case r'email_taken':
          return ErrorErrorEnum.emailTaken;
        case r'unauthenticated':
          return ErrorErrorEnum.unauthenticated;
        case r'wrong_credentials':
          return ErrorErrorEnum.wrongCredentials;
        case r'rate_limited':
          return ErrorErrorEnum.rateLimited;
        case r'invalid_challenge':
          return ErrorErrorEnum.invalidChallenge;
        case r'wrong_code':
          return ErrorErrorEnum.wrongCode;
        case r'already_enrolled':
          return ErrorErrorEnum.alreadyEnrolled;
        case r'not_enrolled':
          return ErrorErrorEnum.notEnrolled;
        case r'csrf':
          return ErrorErrorEnum.csrf;
        case r'forbidden':
          return ErrorErrorEnum.forbidden;
        case r'invalid_role':
          return ErrorErrorEnum.invalidRole;
        case r'invalid_name':
          return ErrorErrorEnum.invalidName;
        case r'invalid_invite':
          return ErrorErrorEnum.invalidInvite;
        case r'invalid_birth_date':
          return ErrorErrorEnum.invalidBirthDate;
        case r'invalid_phone':
          return ErrorErrorEnum.invalidPhone;
        case r'not_found':
          return ErrorErrorEnum.notFound;
        case r'version_conflict':
          return ErrorErrorEnum.versionConflict;
        case r'consent_active':
          return ErrorErrorEnum.consentActive;
        case r'consent_inactive':
          return ErrorErrorEnum.consentInactive;
        case r'profile_required':
          return ErrorErrorEnum.profileRequired;
        case r'internal error':
          return ErrorErrorEnum.internalError;
        case r'timeout':
          return ErrorErrorEnum.timeout;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ErrorErrorEnumTypeTransformer? _instance;
}
