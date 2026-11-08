//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ConsultMessage {
  /// Returns a new [ConsultMessage] instance.
  ConsultMessage({
    required this.type,
    required this.version,
    required this.payload,
  });

  ConsultMessageTypeEnum type;

  ConsultMessageVersionEnum version;

  DraftEvent payload;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ConsultMessage &&
          other.type == type &&
          other.version == version &&
          other.payload == payload;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (type.hashCode) + (version.hashCode) + (payload.hashCode);

  @override
  String toString() =>
      'ConsultMessage[type=$type, version=$version, payload=$payload]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'type'] = this.type;
    json[r'version'] = this.version;
    json[r'payload'] = this.payload;
    return json;
  }

  /// Returns a new [ConsultMessage] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ConsultMessage? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'type'),
            'Required key "ConsultMessage[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "ConsultMessage[type]" has a null value in JSON.');
        assert(json.containsKey(r'version'),
            'Required key "ConsultMessage[version]" is missing from JSON.');
        assert(json[r'version'] != null,
            'Required key "ConsultMessage[version]" has a null value in JSON.');
        assert(json.containsKey(r'payload'),
            'Required key "ConsultMessage[payload]" is missing from JSON.');
        assert(json[r'payload'] != null,
            'Required key "ConsultMessage[payload]" has a null value in JSON.');
        return true;
      }());

      return ConsultMessage(
        type: ConsultMessageTypeEnum.fromJson(json[r'type'])!,
        version: ConsultMessageVersionEnum.fromJson(json[r'version'])!,
        payload: DraftEvent.fromJson(json[r'payload'])!,
      );
    }
    return null;
  }

  static List<ConsultMessage> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ConsultMessage>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ConsultMessage.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ConsultMessage> mapFromJson(dynamic json) {
    final map = <String, ConsultMessage>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ConsultMessage.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ConsultMessage-objects as value to a dart map
  static Map<String, List<ConsultMessage>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ConsultMessage>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ConsultMessage.listFromJson(
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

enum ConsultMessageTypeEnum {
  transcript._(r'transcript'),
  suggestion._(r'suggestion'),
  draft._(r'draft'),
  ;

  /// Instantiate a new enum with the provided value.
  const ConsultMessageTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ConsultMessageTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ConsultMessageTypeEnum? fromJson(dynamic value) =>
      ConsultMessageTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ConsultMessageTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ConsultMessageTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ConsultMessageTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ConsultMessageTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ConsultMessageTypeEnum] to String,
/// and [decode] dynamic data back to [ConsultMessageTypeEnum].
class ConsultMessageTypeEnumTypeTransformer {
  factory ConsultMessageTypeEnumTypeTransformer() =>
      _instance ??= const ConsultMessageTypeEnumTypeTransformer._();

  const ConsultMessageTypeEnumTypeTransformer._();

  String encode(ConsultMessageTypeEnum data) => data._value;

  /// Returns the instance of [ConsultMessageTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ConsultMessageTypeEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ConsultMessageTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'transcript':
          return ConsultMessageTypeEnum.transcript;
        case r'suggestion':
          return ConsultMessageTypeEnum.suggestion;
        case r'draft':
          return ConsultMessageTypeEnum.draft;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ConsultMessageTypeEnumTypeTransformer? _instance;
}

enum ConsultMessageVersionEnum {
  number1._(1),
  ;

  /// Instantiate a new enum with the provided value.
  const ConsultMessageVersionEnum._(this._value);

  /// The underlying value of this enum member.
  final int _value;

  @override
  String toString() => _value.toString();

  /// Encodes this enum as a value suitable for JSON.
  int toJson() => _value;

  /// Returns the instance of [ConsultMessageVersionEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ConsultMessageVersionEnum? fromJson(dynamic value) =>
      ConsultMessageVersionEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ConsultMessageVersionEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ConsultMessageVersionEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ConsultMessageVersionEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ConsultMessageVersionEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ConsultMessageVersionEnum] to int,
/// and [decode] dynamic data back to [ConsultMessageVersionEnum].
class ConsultMessageVersionEnumTypeTransformer {
  factory ConsultMessageVersionEnumTypeTransformer() =>
      _instance ??= const ConsultMessageVersionEnumTypeTransformer._();

  const ConsultMessageVersionEnumTypeTransformer._();

  int encode(ConsultMessageVersionEnum data) => data._value;

  /// Returns the instance of [ConsultMessageVersionEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ConsultMessageVersionEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ConsultMessageVersionEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case 1:
          return ConsultMessageVersionEnum.number1;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ConsultMessageVersionEnumTypeTransformer? _instance;
}
