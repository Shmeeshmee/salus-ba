//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Consent {
  /// Returns a new [Consent] instance.
  Consent({
    required this.id,
    required this.organizationId,
    required this.kind,
    required this.status,
    required this.dateTime,
    this.revokedAt,
  });

  String id;

  String organizationId;

  ConsentKind kind;

  ConsentStatusEnum status;

  /// When it was granted.
  DateTime dateTime;

  /// When it was revoked; absent while active.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? revokedAt;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Consent &&
          other.id == id &&
          other.organizationId == organizationId &&
          other.kind == kind &&
          other.status == status &&
          other.dateTime == dateTime &&
          other.revokedAt == revokedAt;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (organizationId.hashCode) +
      (kind.hashCode) +
      (status.hashCode) +
      (dateTime.hashCode) +
      (revokedAt == null ? 0 : revokedAt!.hashCode);

  @override
  String toString() =>
      'Consent[id=$id, organizationId=$organizationId, kind=$kind, status=$status, dateTime=$dateTime, revokedAt=$revokedAt]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'organizationId'] = this.organizationId;
    json[r'kind'] = this.kind;
    json[r'status'] = this.status;
    json[r'dateTime'] = this.dateTime.toUtc().toIso8601String();
    if (this.revokedAt != null) {
      json[r'revokedAt'] = this.revokedAt!.toUtc().toIso8601String();
    } else {
      json[r'revokedAt'] = null;
    }
    return json;
  }

  /// Returns a new [Consent] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Consent? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "Consent[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "Consent[id]" has a null value in JSON.');
        assert(json.containsKey(r'organizationId'),
            'Required key "Consent[organizationId]" is missing from JSON.');
        assert(json[r'organizationId'] != null,
            'Required key "Consent[organizationId]" has a null value in JSON.');
        assert(json.containsKey(r'kind'),
            'Required key "Consent[kind]" is missing from JSON.');
        assert(json[r'kind'] != null,
            'Required key "Consent[kind]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "Consent[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "Consent[status]" has a null value in JSON.');
        assert(json.containsKey(r'dateTime'),
            'Required key "Consent[dateTime]" is missing from JSON.');
        assert(json[r'dateTime'] != null,
            'Required key "Consent[dateTime]" has a null value in JSON.');
        return true;
      }());

      return Consent(
        id: mapValueOfType<String>(json, r'id')!,
        organizationId: mapValueOfType<String>(json, r'organizationId')!,
        kind: ConsentKind.fromJson(json[r'kind'])!,
        status: ConsentStatusEnum.fromJson(json[r'status'])!,
        dateTime: mapDateTime(json, r'dateTime', r'')!,
        revokedAt: mapDateTime(json, r'revokedAt', r''),
      );
    }
    return null;
  }

  static List<Consent> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Consent>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Consent.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Consent> mapFromJson(dynamic json) {
    final map = <String, Consent>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Consent.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Consent-objects as value to a dart map
  static Map<String, List<Consent>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Consent>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Consent.listFromJson(
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
    'organizationId',
    'kind',
    'status',
    'dateTime',
  };
}

enum ConsentStatusEnum {
  active._(r'active'),
  inactive._(r'inactive'),
  ;

  /// Instantiate a new enum with the provided value.
  const ConsentStatusEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ConsentStatusEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ConsentStatusEnum? fromJson(dynamic value) =>
      ConsentStatusEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ConsentStatusEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ConsentStatusEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ConsentStatusEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ConsentStatusEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ConsentStatusEnum] to String,
/// and [decode] dynamic data back to [ConsentStatusEnum].
class ConsentStatusEnumTypeTransformer {
  factory ConsentStatusEnumTypeTransformer() =>
      _instance ??= const ConsentStatusEnumTypeTransformer._();

  const ConsentStatusEnumTypeTransformer._();

  String encode(ConsentStatusEnum data) => data._value;

  /// Returns the instance of [ConsentStatusEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ConsentStatusEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ConsentStatusEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'active':
          return ConsentStatusEnum.active;
        case r'inactive':
          return ConsentStatusEnum.inactive;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ConsentStatusEnumTypeTransformer? _instance;
}
