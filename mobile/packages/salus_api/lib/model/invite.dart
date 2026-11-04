//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Invite {
  /// Returns a new [Invite] instance.
  Invite({
    required this.id,
    required this.email,
    required this.role,
    this.givenName,
    this.familyName,
    required this.expiresAt,
  });

  String id;

  String email;

  InviteRoleEnum role;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? givenName;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? familyName;

  DateTime expiresAt;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Invite &&
          other.id == id &&
          other.email == email &&
          other.role == role &&
          other.givenName == givenName &&
          other.familyName == familyName &&
          other.expiresAt == expiresAt;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (email.hashCode) +
      (role.hashCode) +
      (givenName == null ? 0 : givenName!.hashCode) +
      (familyName == null ? 0 : familyName!.hashCode) +
      (expiresAt.hashCode);

  @override
  String toString() =>
      'Invite[id=$id, email=$email, role=$role, givenName=$givenName, familyName=$familyName, expiresAt=$expiresAt]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'email'] = this.email;
    json[r'role'] = this.role;
    if (this.givenName != null) {
      json[r'givenName'] = this.givenName;
    } else {
      json[r'givenName'] = null;
    }
    if (this.familyName != null) {
      json[r'familyName'] = this.familyName;
    } else {
      json[r'familyName'] = null;
    }
    json[r'expiresAt'] = this.expiresAt.toUtc().toIso8601String();
    return json;
  }

  /// Returns a new [Invite] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Invite? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "Invite[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "Invite[id]" has a null value in JSON.');
        assert(json.containsKey(r'email'),
            'Required key "Invite[email]" is missing from JSON.');
        assert(json[r'email'] != null,
            'Required key "Invite[email]" has a null value in JSON.');
        assert(json.containsKey(r'role'),
            'Required key "Invite[role]" is missing from JSON.');
        assert(json[r'role'] != null,
            'Required key "Invite[role]" has a null value in JSON.');
        assert(json.containsKey(r'expiresAt'),
            'Required key "Invite[expiresAt]" is missing from JSON.');
        assert(json[r'expiresAt'] != null,
            'Required key "Invite[expiresAt]" has a null value in JSON.');
        return true;
      }());

      return Invite(
        id: mapValueOfType<String>(json, r'id')!,
        email: mapValueOfType<String>(json, r'email')!,
        role: InviteRoleEnum.fromJson(json[r'role'])!,
        givenName: mapValueOfType<String>(json, r'givenName'),
        familyName: mapValueOfType<String>(json, r'familyName'),
        expiresAt: mapDateTime(json, r'expiresAt', r'')!,
      );
    }
    return null;
  }

  static List<Invite> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Invite>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Invite.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Invite> mapFromJson(dynamic json) {
    final map = <String, Invite>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Invite.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Invite-objects as value to a dart map
  static Map<String, List<Invite>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Invite>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Invite.listFromJson(
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
    'expiresAt',
  };
}

enum InviteRoleEnum {
  admin._(r'admin'),
  practitioner._(r'practitioner'),
  ;

  /// Instantiate a new enum with the provided value.
  const InviteRoleEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [InviteRoleEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static InviteRoleEnum? fromJson(dynamic value) =>
      InviteRoleEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [InviteRoleEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<InviteRoleEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <InviteRoleEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = InviteRoleEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [InviteRoleEnum] to String,
/// and [decode] dynamic data back to [InviteRoleEnum].
class InviteRoleEnumTypeTransformer {
  factory InviteRoleEnumTypeTransformer() =>
      _instance ??= const InviteRoleEnumTypeTransformer._();

  const InviteRoleEnumTypeTransformer._();

  String encode(InviteRoleEnum data) => data._value;

  /// Returns the instance of [InviteRoleEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  InviteRoleEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is InviteRoleEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'admin':
          return InviteRoleEnum.admin;
        case r'practitioner':
          return InviteRoleEnum.practitioner;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static InviteRoleEnumTypeTransformer? _instance;
}
