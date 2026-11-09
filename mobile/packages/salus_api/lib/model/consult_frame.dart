//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ConsultFrame {
  /// Returns a new [ConsultFrame] instance.
  ConsultFrame({
    required this.type,
    required this.version,
    required this.payload,
  });

  ConsultFrameTypeEnum type;

  ConsultFrameVersionEnum version;

  Cursor payload;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ConsultFrame &&
          other.type == type &&
          other.version == version &&
          other.payload == payload;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (type.hashCode) + (version.hashCode) + (payload.hashCode);

  @override
  String toString() =>
      'ConsultFrame[type=$type, version=$version, payload=$payload]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'type'] = this.type;
    json[r'version'] = this.version;
    json[r'payload'] = this.payload;
    return json;
  }

  /// Returns a new [ConsultFrame] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ConsultFrame? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'type'),
            'Required key "ConsultFrame[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "ConsultFrame[type]" has a null value in JSON.');
        assert(json.containsKey(r'version'),
            'Required key "ConsultFrame[version]" is missing from JSON.');
        assert(json[r'version'] != null,
            'Required key "ConsultFrame[version]" has a null value in JSON.');
        assert(json.containsKey(r'payload'),
            'Required key "ConsultFrame[payload]" is missing from JSON.');
        assert(json[r'payload'] != null,
            'Required key "ConsultFrame[payload]" has a null value in JSON.');
        return true;
      }());

      return ConsultFrame(
        type: ConsultFrameTypeEnum.fromJson(json[r'type'])!,
        version: ConsultFrameVersionEnum.fromJson(json[r'version'])!,
        payload: Cursor.fromJson(json[r'payload'])!,
      );
    }
    return null;
  }

  static List<ConsultFrame> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ConsultFrame>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ConsultFrame.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ConsultFrame> mapFromJson(dynamic json) {
    final map = <String, ConsultFrame>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ConsultFrame.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ConsultFrame-objects as value to a dart map
  static Map<String, List<ConsultFrame>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ConsultFrame>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ConsultFrame.listFromJson(
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

enum ConsultFrameTypeEnum {
  transcript._(r'transcript'),
  suggestion._(r'suggestion'),
  draft._(r'draft'),
  cursor._(r'cursor'),
  ;

  /// Instantiate a new enum with the provided value.
  const ConsultFrameTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ConsultFrameTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ConsultFrameTypeEnum? fromJson(dynamic value) =>
      ConsultFrameTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ConsultFrameTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ConsultFrameTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ConsultFrameTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ConsultFrameTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ConsultFrameTypeEnum] to String,
/// and [decode] dynamic data back to [ConsultFrameTypeEnum].
class ConsultFrameTypeEnumTypeTransformer {
  factory ConsultFrameTypeEnumTypeTransformer() =>
      _instance ??= const ConsultFrameTypeEnumTypeTransformer._();

  const ConsultFrameTypeEnumTypeTransformer._();

  String encode(ConsultFrameTypeEnum data) => data._value;

  /// Returns the instance of [ConsultFrameTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ConsultFrameTypeEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ConsultFrameTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'transcript':
          return ConsultFrameTypeEnum.transcript;
        case r'suggestion':
          return ConsultFrameTypeEnum.suggestion;
        case r'draft':
          return ConsultFrameTypeEnum.draft;
        case r'cursor':
          return ConsultFrameTypeEnum.cursor;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ConsultFrameTypeEnumTypeTransformer? _instance;
}

enum ConsultFrameVersionEnum {
  number1._(1),
  ;

  /// Instantiate a new enum with the provided value.
  const ConsultFrameVersionEnum._(this._value);

  /// The underlying value of this enum member.
  final int _value;

  @override
  String toString() => _value.toString();

  /// Encodes this enum as a value suitable for JSON.
  int toJson() => _value;

  /// Returns the instance of [ConsultFrameVersionEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ConsultFrameVersionEnum? fromJson(dynamic value) =>
      ConsultFrameVersionEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ConsultFrameVersionEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ConsultFrameVersionEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ConsultFrameVersionEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ConsultFrameVersionEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ConsultFrameVersionEnum] to int,
/// and [decode] dynamic data back to [ConsultFrameVersionEnum].
class ConsultFrameVersionEnumTypeTransformer {
  factory ConsultFrameVersionEnumTypeTransformer() =>
      _instance ??= const ConsultFrameVersionEnumTypeTransformer._();

  const ConsultFrameVersionEnumTypeTransformer._();

  int encode(ConsultFrameVersionEnum data) => data._value;

  /// Returns the instance of [ConsultFrameVersionEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ConsultFrameVersionEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ConsultFrameVersionEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case 1:
          return ConsultFrameVersionEnum.number1;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ConsultFrameVersionEnumTypeTransformer? _instance;
}
