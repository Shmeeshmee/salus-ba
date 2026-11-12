//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SendAckFrame {
  /// Returns a new [SendAckFrame] instance.
  SendAckFrame({
    required this.type,
    required this.version,
    required this.payload,
  });

  SendAckFrameTypeEnum type;

  SendAckFrameVersionEnum version;

  SendAck payload;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SendAckFrame &&
          other.type == type &&
          other.version == version &&
          other.payload == payload;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (type.hashCode) + (version.hashCode) + (payload.hashCode);

  @override
  String toString() =>
      'SendAckFrame[type=$type, version=$version, payload=$payload]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'type'] = this.type;
    json[r'version'] = this.version;
    json[r'payload'] = this.payload;
    return json;
  }

  /// Returns a new [SendAckFrame] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SendAckFrame? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'type'),
            'Required key "SendAckFrame[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "SendAckFrame[type]" has a null value in JSON.');
        assert(json.containsKey(r'version'),
            'Required key "SendAckFrame[version]" is missing from JSON.');
        assert(json[r'version'] != null,
            'Required key "SendAckFrame[version]" has a null value in JSON.');
        assert(json.containsKey(r'payload'),
            'Required key "SendAckFrame[payload]" is missing from JSON.');
        assert(json[r'payload'] != null,
            'Required key "SendAckFrame[payload]" has a null value in JSON.');
        return true;
      }());

      return SendAckFrame(
        type: SendAckFrameTypeEnum.fromJson(json[r'type'])!,
        version: SendAckFrameVersionEnum.fromJson(json[r'version'])!,
        payload: SendAck.fromJson(json[r'payload'])!,
      );
    }
    return null;
  }

  static List<SendAckFrame> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SendAckFrame>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SendAckFrame.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SendAckFrame> mapFromJson(dynamic json) {
    final map = <String, SendAckFrame>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SendAckFrame.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SendAckFrame-objects as value to a dart map
  static Map<String, List<SendAckFrame>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SendAckFrame>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SendAckFrame.listFromJson(
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

enum SendAckFrameTypeEnum {
  sendAck._(r'send_ack'),
  ;

  /// Instantiate a new enum with the provided value.
  const SendAckFrameTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [SendAckFrameTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static SendAckFrameTypeEnum? fromJson(dynamic value) =>
      SendAckFrameTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [SendAckFrameTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<SendAckFrameTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SendAckFrameTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SendAckFrameTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [SendAckFrameTypeEnum] to String,
/// and [decode] dynamic data back to [SendAckFrameTypeEnum].
class SendAckFrameTypeEnumTypeTransformer {
  factory SendAckFrameTypeEnumTypeTransformer() =>
      _instance ??= const SendAckFrameTypeEnumTypeTransformer._();

  const SendAckFrameTypeEnumTypeTransformer._();

  String encode(SendAckFrameTypeEnum data) => data._value;

  /// Returns the instance of [SendAckFrameTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  SendAckFrameTypeEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is SendAckFrameTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'send_ack':
          return SendAckFrameTypeEnum.sendAck;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static SendAckFrameTypeEnumTypeTransformer? _instance;
}

enum SendAckFrameVersionEnum {
  number1._(1),
  ;

  /// Instantiate a new enum with the provided value.
  const SendAckFrameVersionEnum._(this._value);

  /// The underlying value of this enum member.
  final int _value;

  @override
  String toString() => _value.toString();

  /// Encodes this enum as a value suitable for JSON.
  int toJson() => _value;

  /// Returns the instance of [SendAckFrameVersionEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static SendAckFrameVersionEnum? fromJson(dynamic value) =>
      SendAckFrameVersionEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [SendAckFrameVersionEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<SendAckFrameVersionEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SendAckFrameVersionEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SendAckFrameVersionEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [SendAckFrameVersionEnum] to int,
/// and [decode] dynamic data back to [SendAckFrameVersionEnum].
class SendAckFrameVersionEnumTypeTransformer {
  factory SendAckFrameVersionEnumTypeTransformer() =>
      _instance ??= const SendAckFrameVersionEnumTypeTransformer._();

  const SendAckFrameVersionEnumTypeTransformer._();

  int encode(SendAckFrameVersionEnum data) => data._value;

  /// Returns the instance of [SendAckFrameVersionEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  SendAckFrameVersionEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is SendAckFrameVersionEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case 1:
          return SendAckFrameVersionEnum.number1;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static SendAckFrameVersionEnumTypeTransformer? _instance;
}
