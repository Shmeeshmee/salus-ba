//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Account {
  /// Returns a new [Account] instance.
  Account({
    required this.id,
    required this.email,
    required this.role,
    this.organizationId,
    required this.mfaEnrolled,
  });

  String id;

  String email;

  AccountRoleEnum role;

  /// The clinic of a staff account; absent for a patient.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? organizationId;

  bool mfaEnrolled;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Account &&
          other.id == id &&
          other.email == email &&
          other.role == role &&
          other.organizationId == organizationId &&
          other.mfaEnrolled == mfaEnrolled;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (email.hashCode) +
      (role.hashCode) +
      (organizationId == null ? 0 : organizationId!.hashCode) +
      (mfaEnrolled.hashCode);

  @override
  String toString() =>
      'Account[id=$id, email=$email, role=$role, organizationId=$organizationId, mfaEnrolled=$mfaEnrolled]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'email'] = this.email;
    json[r'role'] = this.role;
    if (this.organizationId != null) {
      json[r'organizationId'] = this.organizationId;
    } else {
      json[r'organizationId'] = null;
    }
    json[r'mfaEnrolled'] = this.mfaEnrolled;
    return json;
  }

  /// Returns a new [Account] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Account? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "Account[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "Account[id]" has a null value in JSON.');
        assert(json.containsKey(r'email'),
            'Required key "Account[email]" is missing from JSON.');
        assert(json[r'email'] != null,
            'Required key "Account[email]" has a null value in JSON.');
        assert(json.containsKey(r'role'),
            'Required key "Account[role]" is missing from JSON.');
        assert(json[r'role'] != null,
            'Required key "Account[role]" has a null value in JSON.');
        assert(json.containsKey(r'mfaEnrolled'),
            'Required key "Account[mfaEnrolled]" is missing from JSON.');
        assert(json[r'mfaEnrolled'] != null,
            'Required key "Account[mfaEnrolled]" has a null value in JSON.');
        return true;
      }());

      return Account(
        id: mapValueOfType<String>(json, r'id')!,
        email: mapValueOfType<String>(json, r'email')!,
        role: AccountRoleEnum.fromJson(json[r'role'])!,
        organizationId: mapValueOfType<String>(json, r'organizationId'),
        mfaEnrolled: mapValueOfType<bool>(json, r'mfaEnrolled')!,
      );
    }
    return null;
  }

  static List<Account> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Account>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Account.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Account> mapFromJson(dynamic json) {
    final map = <String, Account>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Account.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Account-objects as value to a dart map
  static Map<String, List<Account>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Account>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Account.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'id',
    'email',
    'role',
    'mfaEnrolled',
  };
}

enum AccountRoleEnum {
  admin._(r'admin'),
  practitioner._(r'practitioner'),
  patient._(r'patient'),
  ;

  /// Instantiate a new enum with the provided value.
  const AccountRoleEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [AccountRoleEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static AccountRoleEnum? fromJson(dynamic value) =>
      AccountRoleEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [AccountRoleEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<AccountRoleEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AccountRoleEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AccountRoleEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [AccountRoleEnum] to String,
/// and [decode] dynamic data back to [AccountRoleEnum].
class AccountRoleEnumTypeTransformer {
  factory AccountRoleEnumTypeTransformer() =>
      _instance ??= const AccountRoleEnumTypeTransformer._();

  const AccountRoleEnumTypeTransformer._();

  String encode(AccountRoleEnum data) => data._value;

  /// Returns the instance of [AccountRoleEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  AccountRoleEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is AccountRoleEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'admin':
          return AccountRoleEnum.admin;
        case r'practitioner':
          return AccountRoleEnum.practitioner;
        case r'patient':
          return AccountRoleEnum.patient;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static AccountRoleEnumTypeTransformer? _instance;
}
