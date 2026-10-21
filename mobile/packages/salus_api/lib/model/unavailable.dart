//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Unavailable {
  /// Returns a new [Unavailable] instance.
  Unavailable({
    required this.status,
    this.failed = const [],
  });

  UnavailableStatusEnum status;

  /// Names of the failing checks.
  List<UnavailableFailedEnum> failed;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Unavailable &&
          other.status == status &&
          _deepEquality.equals(other.failed, failed);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (status.hashCode) + (failed.hashCode);

  @override
  String toString() => 'Unavailable[status=$status, failed=$failed]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'status'] = this.status;
    json[r'failed'] = this.failed;
    return json;
  }

  /// Returns a new [Unavailable] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Unavailable? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'status'),
            'Required key "Unavailable[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "Unavailable[status]" has a null value in JSON.');
        assert(json.containsKey(r'failed'),
            'Required key "Unavailable[failed]" is missing from JSON.');
        assert(json[r'failed'] != null,
            'Required key "Unavailable[failed]" has a null value in JSON.');
        return true;
      }());

      return Unavailable(
        status: UnavailableStatusEnum.fromJson(json[r'status'])!,
        failed: UnavailableFailedEnum.listFromJson(json[r'failed']),
      );
    }
    return null;
  }

  static List<Unavailable> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Unavailable>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Unavailable.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Unavailable> mapFromJson(dynamic json) {
    final map = <String, Unavailable>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Unavailable.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Unavailable-objects as value to a dart map
  static Map<String, List<Unavailable>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Unavailable>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Unavailable.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'status',
    'failed',
  };
}

enum UnavailableStatusEnum {
  unavailable._(r'unavailable'),
  ;

  /// Instantiate a new enum with the provided value.
  const UnavailableStatusEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [UnavailableStatusEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static UnavailableStatusEnum? fromJson(dynamic value) =>
      UnavailableStatusEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [UnavailableStatusEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<UnavailableStatusEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <UnavailableStatusEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = UnavailableStatusEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [UnavailableStatusEnum] to String,
/// and [decode] dynamic data back to [UnavailableStatusEnum].
class UnavailableStatusEnumTypeTransformer {
  factory UnavailableStatusEnumTypeTransformer() =>
      _instance ??= const UnavailableStatusEnumTypeTransformer._();

  const UnavailableStatusEnumTypeTransformer._();

  String encode(UnavailableStatusEnum data) => data._value;

  /// Returns the instance of [UnavailableStatusEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  UnavailableStatusEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is UnavailableStatusEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'unavailable':
          return UnavailableStatusEnum.unavailable;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static UnavailableStatusEnumTypeTransformer? _instance;
}

enum UnavailableFailedEnum {
  postgres._(r'postgres'),
  redis._(r'redis'),
  ;

  /// Instantiate a new enum with the provided value.
  const UnavailableFailedEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [UnavailableFailedEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static UnavailableFailedEnum? fromJson(dynamic value) =>
      UnavailableFailedEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [UnavailableFailedEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<UnavailableFailedEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <UnavailableFailedEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = UnavailableFailedEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [UnavailableFailedEnum] to String,
/// and [decode] dynamic data back to [UnavailableFailedEnum].
class UnavailableFailedEnumTypeTransformer {
  factory UnavailableFailedEnumTypeTransformer() =>
      _instance ??= const UnavailableFailedEnumTypeTransformer._();

  const UnavailableFailedEnumTypeTransformer._();

  String encode(UnavailableFailedEnum data) => data._value;

  /// Returns the instance of [UnavailableFailedEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  UnavailableFailedEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is UnavailableFailedEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'postgres':
          return UnavailableFailedEnum.postgres;
        case r'redis':
          return UnavailableFailedEnum.redis;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static UnavailableFailedEnumTypeTransformer? _instance;
}
