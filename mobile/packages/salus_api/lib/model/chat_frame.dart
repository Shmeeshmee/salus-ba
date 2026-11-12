//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ChatFrame {
  /// Returns a new [ChatFrame] instance.
  ChatFrame({
    required this.type,
    required this.version,
    required this.payload,
  });

  ChatFrameTypeEnum type;

  ChatFrameVersionEnum version;

  ChatCommandError payload;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ChatFrame &&
          other.type == type &&
          other.version == version &&
          other.payload == payload;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (type.hashCode) + (version.hashCode) + (payload.hashCode);

  @override
  String toString() =>
      'ChatFrame[type=$type, version=$version, payload=$payload]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'type'] = this.type;
    json[r'version'] = this.version;
    json[r'payload'] = this.payload;
    return json;
  }

  /// Returns a new [ChatFrame] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ChatFrame? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'type'),
            'Required key "ChatFrame[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "ChatFrame[type]" has a null value in JSON.');
        assert(json.containsKey(r'version'),
            'Required key "ChatFrame[version]" is missing from JSON.');
        assert(json[r'version'] != null,
            'Required key "ChatFrame[version]" has a null value in JSON.');
        assert(json.containsKey(r'payload'),
            'Required key "ChatFrame[payload]" is missing from JSON.');
        assert(json[r'payload'] != null,
            'Required key "ChatFrame[payload]" has a null value in JSON.');
        return true;
      }());

      return ChatFrame(
        type: ChatFrameTypeEnum.fromJson(json[r'type'])!,
        version: ChatFrameVersionEnum.fromJson(json[r'version'])!,
        payload: ChatCommandError.fromJson(json[r'payload'])!,
      );
    }
    return null;
  }

  static List<ChatFrame> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatFrame>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatFrame.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ChatFrame> mapFromJson(dynamic json) {
    final map = <String, ChatFrame>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ChatFrame.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ChatFrame-objects as value to a dart map
  static Map<String, List<ChatFrame>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ChatFrame>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ChatFrame.listFromJson(
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

enum ChatFrameTypeEnum {
  message._(r'message'),
  messageEdited._(r'message_edited'),
  sendAck._(r'send_ack'),
  commandError._(r'command_error'),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatFrameTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ChatFrameTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatFrameTypeEnum? fromJson(dynamic value) =>
      ChatFrameTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatFrameTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatFrameTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatFrameTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatFrameTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatFrameTypeEnum] to String,
/// and [decode] dynamic data back to [ChatFrameTypeEnum].
class ChatFrameTypeEnumTypeTransformer {
  factory ChatFrameTypeEnumTypeTransformer() =>
      _instance ??= const ChatFrameTypeEnumTypeTransformer._();

  const ChatFrameTypeEnumTypeTransformer._();

  String encode(ChatFrameTypeEnum data) => data._value;

  /// Returns the instance of [ChatFrameTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatFrameTypeEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ChatFrameTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'message':
          return ChatFrameTypeEnum.message;
        case r'message_edited':
          return ChatFrameTypeEnum.messageEdited;
        case r'send_ack':
          return ChatFrameTypeEnum.sendAck;
        case r'command_error':
          return ChatFrameTypeEnum.commandError;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatFrameTypeEnumTypeTransformer? _instance;
}

enum ChatFrameVersionEnum {
  number1._(1),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatFrameVersionEnum._(this._value);

  /// The underlying value of this enum member.
  final int _value;

  @override
  String toString() => _value.toString();

  /// Encodes this enum as a value suitable for JSON.
  int toJson() => _value;

  /// Returns the instance of [ChatFrameVersionEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatFrameVersionEnum? fromJson(dynamic value) =>
      ChatFrameVersionEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatFrameVersionEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatFrameVersionEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatFrameVersionEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatFrameVersionEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatFrameVersionEnum] to int,
/// and [decode] dynamic data back to [ChatFrameVersionEnum].
class ChatFrameVersionEnumTypeTransformer {
  factory ChatFrameVersionEnumTypeTransformer() =>
      _instance ??= const ChatFrameVersionEnumTypeTransformer._();

  const ChatFrameVersionEnumTypeTransformer._();

  int encode(ChatFrameVersionEnum data) => data._value;

  /// Returns the instance of [ChatFrameVersionEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatFrameVersionEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ChatFrameVersionEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case 1:
          return ChatFrameVersionEnum.number1;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatFrameVersionEnumTypeTransformer? _instance;
}
