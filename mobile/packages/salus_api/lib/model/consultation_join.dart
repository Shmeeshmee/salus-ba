//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ConsultationJoin {
  /// Returns a new [ConsultationJoin] instance.
  ConsultationJoin({
    required this.url,
    required this.token,
    required this.expiresAt,
    required this.peerName,
    required this.role,
  });

  /// The LiveKit server the client connects to (ws:// locally, wss:// deployed).
  String url;

  /// The LiveKit join token, valid 5 minutes for joining.
  String token;

  DateTime expiresAt;

  /// The other person's display name; empty when the caller may not see it (the practitioner after a revocation, or a practitioner not in the public directory).
  String peerName;

  /// The caller's role in the consultation.
  ConsultationJoinRoleEnum role;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ConsultationJoin &&
          other.url == url &&
          other.token == token &&
          other.expiresAt == expiresAt &&
          other.peerName == peerName &&
          other.role == role;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (url.hashCode) +
      (token.hashCode) +
      (expiresAt.hashCode) +
      (peerName.hashCode) +
      (role.hashCode);

  @override
  String toString() =>
      'ConsultationJoin[url=$url, token=$token, expiresAt=$expiresAt, peerName=$peerName, role=$role]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'url'] = this.url;
    json[r'token'] = this.token;
    json[r'expiresAt'] = this.expiresAt.toUtc().toIso8601String();
    json[r'peerName'] = this.peerName;
    json[r'role'] = this.role;
    return json;
  }

  /// Returns a new [ConsultationJoin] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ConsultationJoin? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'url'),
            'Required key "ConsultationJoin[url]" is missing from JSON.');
        assert(json[r'url'] != null,
            'Required key "ConsultationJoin[url]" has a null value in JSON.');
        assert(json.containsKey(r'token'),
            'Required key "ConsultationJoin[token]" is missing from JSON.');
        assert(json[r'token'] != null,
            'Required key "ConsultationJoin[token]" has a null value in JSON.');
        assert(json.containsKey(r'expiresAt'),
            'Required key "ConsultationJoin[expiresAt]" is missing from JSON.');
        assert(json[r'expiresAt'] != null,
            'Required key "ConsultationJoin[expiresAt]" has a null value in JSON.');
        assert(json.containsKey(r'peerName'),
            'Required key "ConsultationJoin[peerName]" is missing from JSON.');
        assert(json[r'peerName'] != null,
            'Required key "ConsultationJoin[peerName]" has a null value in JSON.');
        assert(json.containsKey(r'role'),
            'Required key "ConsultationJoin[role]" is missing from JSON.');
        assert(json[r'role'] != null,
            'Required key "ConsultationJoin[role]" has a null value in JSON.');
        return true;
      }());

      return ConsultationJoin(
        url: mapValueOfType<String>(json, r'url')!,
        token: mapValueOfType<String>(json, r'token')!,
        expiresAt: mapDateTime(json, r'expiresAt', r'')!,
        peerName: mapValueOfType<String>(json, r'peerName')!,
        role: ConsultationJoinRoleEnum.fromJson(json[r'role'])!,
      );
    }
    return null;
  }

  static List<ConsultationJoin> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ConsultationJoin>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ConsultationJoin.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ConsultationJoin> mapFromJson(dynamic json) {
    final map = <String, ConsultationJoin>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ConsultationJoin.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ConsultationJoin-objects as value to a dart map
  static Map<String, List<ConsultationJoin>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ConsultationJoin>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ConsultationJoin.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'url',
    'token',
    'expiresAt',
    'peerName',
    'role',
  };
}

/// The caller's role in the consultation.
enum ConsultationJoinRoleEnum {
  patient._(r'patient'),
  practitioner._(r'practitioner'),
  ;

  /// Instantiate a new enum with the provided value.
  const ConsultationJoinRoleEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ConsultationJoinRoleEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ConsultationJoinRoleEnum? fromJson(dynamic value) =>
      ConsultationJoinRoleEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ConsultationJoinRoleEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ConsultationJoinRoleEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ConsultationJoinRoleEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ConsultationJoinRoleEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ConsultationJoinRoleEnum] to String,
/// and [decode] dynamic data back to [ConsultationJoinRoleEnum].
class ConsultationJoinRoleEnumTypeTransformer {
  factory ConsultationJoinRoleEnumTypeTransformer() =>
      _instance ??= const ConsultationJoinRoleEnumTypeTransformer._();

  const ConsultationJoinRoleEnumTypeTransformer._();

  String encode(ConsultationJoinRoleEnum data) => data._value;

  /// Returns the instance of [ConsultationJoinRoleEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ConsultationJoinRoleEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ConsultationJoinRoleEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'patient':
          return ConsultationJoinRoleEnum.patient;
        case r'practitioner':
          return ConsultationJoinRoleEnum.practitioner;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ConsultationJoinRoleEnumTypeTransformer? _instance;
}
