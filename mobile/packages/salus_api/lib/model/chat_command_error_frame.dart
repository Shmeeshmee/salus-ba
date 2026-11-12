//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ChatCommandErrorFrame {
  /// Returns a new [ChatCommandErrorFrame] instance.
  ChatCommandErrorFrame({
    required this.type,
    required this.version,
    required this.payload,
  });

  ChatCommandErrorFrameTypeEnum type;

  ChatCommandErrorFrameVersionEnum version;

  ChatCommandError payload;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ChatCommandErrorFrame &&
          other.type == type &&
          other.version == version &&
          other.payload == payload;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (type.hashCode) + (version.hashCode) + (payload.hashCode);

  @override
  String toString() =>
      'ChatCommandErrorFrame[type=$type, version=$version, payload=$payload]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'type'] = this.type;
    json[r'version'] = this.version;
    json[r'payload'] = this.payload;
    return json;
  }

  /// Returns a new [ChatCommandErrorFrame] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ChatCommandErrorFrame? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'type'),
            'Required key "ChatCommandErrorFrame[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "ChatCommandErrorFrame[type]" has a null value in JSON.');
        assert(json.containsKey(r'version'),
            'Required key "ChatCommandErrorFrame[version]" is missing from JSON.');
        assert(json[r'version'] != null,
            'Required key "ChatCommandErrorFrame[version]" has a null value in JSON.');
        assert(json.containsKey(r'payload'),
            'Required key "ChatCommandErrorFrame[payload]" is missing from JSON.');
        assert(json[r'payload'] != null,
            'Required key "ChatCommandErrorFrame[payload]" has a null value in JSON.');
        return true;
      }());

      return ChatCommandErrorFrame(
        type: ChatCommandErrorFrameTypeEnum.fromJson(json[r'type'])!,
        version: ChatCommandErrorFrameVersionEnum.fromJson(json[r'version'])!,
        payload: ChatCommandError.fromJson(json[r'payload'])!,
      );
    }
    return null;
  }

  static List<ChatCommandErrorFrame> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatCommandErrorFrame>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatCommandErrorFrame.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ChatCommandErrorFrame> mapFromJson(dynamic json) {
    final map = <String, ChatCommandErrorFrame>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ChatCommandErrorFrame.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ChatCommandErrorFrame-objects as value to a dart map
  static Map<String, List<ChatCommandErrorFrame>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ChatCommandErrorFrame>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ChatCommandErrorFrame.listFromJson(
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

enum ChatCommandErrorFrameTypeEnum {
  commandError._(r'command_error'),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatCommandErrorFrameTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ChatCommandErrorFrameTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatCommandErrorFrameTypeEnum? fromJson(dynamic value) =>
      ChatCommandErrorFrameTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatCommandErrorFrameTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatCommandErrorFrameTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatCommandErrorFrameTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatCommandErrorFrameTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatCommandErrorFrameTypeEnum] to String,
/// and [decode] dynamic data back to [ChatCommandErrorFrameTypeEnum].
class ChatCommandErrorFrameTypeEnumTypeTransformer {
  factory ChatCommandErrorFrameTypeEnumTypeTransformer() =>
      _instance ??= const ChatCommandErrorFrameTypeEnumTypeTransformer._();

  const ChatCommandErrorFrameTypeEnumTypeTransformer._();

  String encode(ChatCommandErrorFrameTypeEnum data) => data._value;

  /// Returns the instance of [ChatCommandErrorFrameTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatCommandErrorFrameTypeEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ChatCommandErrorFrameTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'command_error':
          return ChatCommandErrorFrameTypeEnum.commandError;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatCommandErrorFrameTypeEnumTypeTransformer? _instance;
}

enum ChatCommandErrorFrameVersionEnum {
  number1._(1),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatCommandErrorFrameVersionEnum._(this._value);

  /// The underlying value of this enum member.
  final int _value;

  @override
  String toString() => _value.toString();

  /// Encodes this enum as a value suitable for JSON.
  int toJson() => _value;

  /// Returns the instance of [ChatCommandErrorFrameVersionEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatCommandErrorFrameVersionEnum? fromJson(dynamic value) =>
      ChatCommandErrorFrameVersionEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatCommandErrorFrameVersionEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatCommandErrorFrameVersionEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatCommandErrorFrameVersionEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatCommandErrorFrameVersionEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatCommandErrorFrameVersionEnum] to int,
/// and [decode] dynamic data back to [ChatCommandErrorFrameVersionEnum].
class ChatCommandErrorFrameVersionEnumTypeTransformer {
  factory ChatCommandErrorFrameVersionEnumTypeTransformer() =>
      _instance ??= const ChatCommandErrorFrameVersionEnumTypeTransformer._();

  const ChatCommandErrorFrameVersionEnumTypeTransformer._();

  int encode(ChatCommandErrorFrameVersionEnum data) => data._value;

  /// Returns the instance of [ChatCommandErrorFrameVersionEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatCommandErrorFrameVersionEnum? decode(dynamic data,
      {bool allowNull = true}) {
    if (data is ChatCommandErrorFrameVersionEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case 1:
          return ChatCommandErrorFrameVersionEnum.number1;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatCommandErrorFrameVersionEnumTypeTransformer? _instance;
}
