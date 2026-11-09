//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class CursorMessage {
  /// Returns a new [CursorMessage] instance.
  CursorMessage({
    required this.type,
    required this.version,
    required this.payload,
  });

  CursorMessageTypeEnum type;

  CursorMessageVersionEnum version;

  Cursor payload;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is CursorMessage &&
          other.type == type &&
          other.version == version &&
          other.payload == payload;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (type.hashCode) + (version.hashCode) + (payload.hashCode);

  @override
  String toString() =>
      'CursorMessage[type=$type, version=$version, payload=$payload]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'type'] = this.type;
    json[r'version'] = this.version;
    json[r'payload'] = this.payload;
    return json;
  }

  /// Returns a new [CursorMessage] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static CursorMessage? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'type'),
            'Required key "CursorMessage[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "CursorMessage[type]" has a null value in JSON.');
        assert(json.containsKey(r'version'),
            'Required key "CursorMessage[version]" is missing from JSON.');
        assert(json[r'version'] != null,
            'Required key "CursorMessage[version]" has a null value in JSON.');
        assert(json.containsKey(r'payload'),
            'Required key "CursorMessage[payload]" is missing from JSON.');
        assert(json[r'payload'] != null,
            'Required key "CursorMessage[payload]" has a null value in JSON.');
        return true;
      }());

      return CursorMessage(
        type: CursorMessageTypeEnum.fromJson(json[r'type'])!,
        version: CursorMessageVersionEnum.fromJson(json[r'version'])!,
        payload: Cursor.fromJson(json[r'payload'])!,
      );
    }
    return null;
  }

  static List<CursorMessage> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <CursorMessage>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = CursorMessage.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, CursorMessage> mapFromJson(dynamic json) {
    final map = <String, CursorMessage>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = CursorMessage.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of CursorMessage-objects as value to a dart map
  static Map<String, List<CursorMessage>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<CursorMessage>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = CursorMessage.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'type',
    'version',
    'payload',
  };
}

enum CursorMessageTypeEnum {
  cursor._(r'cursor'),
  ;

  /// Instantiate a new enum with the provided value.
  const CursorMessageTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [CursorMessageTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static CursorMessageTypeEnum? fromJson(dynamic value) =>
      CursorMessageTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [CursorMessageTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<CursorMessageTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <CursorMessageTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = CursorMessageTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [CursorMessageTypeEnum] to String,
/// and [decode] dynamic data back to [CursorMessageTypeEnum].
class CursorMessageTypeEnumTypeTransformer {
  factory CursorMessageTypeEnumTypeTransformer() =>
      _instance ??= const CursorMessageTypeEnumTypeTransformer._();

  const CursorMessageTypeEnumTypeTransformer._();

  String encode(CursorMessageTypeEnum data) => data._value;

  /// Returns the instance of [CursorMessageTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  CursorMessageTypeEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is CursorMessageTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'cursor':
          return CursorMessageTypeEnum.cursor;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static CursorMessageTypeEnumTypeTransformer? _instance;
}

enum CursorMessageVersionEnum {
  number1._(1),
  ;

  /// Instantiate a new enum with the provided value.
  const CursorMessageVersionEnum._(this._value);

  /// The underlying value of this enum member.
  final int _value;

  @override
  String toString() => _value.toString();

  /// Encodes this enum as a value suitable for JSON.
  int toJson() => _value;

  /// Returns the instance of [CursorMessageVersionEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static CursorMessageVersionEnum? fromJson(dynamic value) =>
      CursorMessageVersionEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [CursorMessageVersionEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<CursorMessageVersionEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <CursorMessageVersionEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = CursorMessageVersionEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [CursorMessageVersionEnum] to int,
/// and [decode] dynamic data back to [CursorMessageVersionEnum].
class CursorMessageVersionEnumTypeTransformer {
  factory CursorMessageVersionEnumTypeTransformer() =>
      _instance ??= const CursorMessageVersionEnumTypeTransformer._();

  const CursorMessageVersionEnumTypeTransformer._();

  int encode(CursorMessageVersionEnum data) => data._value;

  /// Returns the instance of [CursorMessageVersionEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  CursorMessageVersionEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is CursorMessageVersionEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case 1:
          return CursorMessageVersionEnum.number1;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static CursorMessageVersionEnumTypeTransformer? _instance;
}
