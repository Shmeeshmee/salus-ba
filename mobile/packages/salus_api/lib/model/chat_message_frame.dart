//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ChatMessageFrame {
  /// Returns a new [ChatMessageFrame] instance.
  ChatMessageFrame({
    required this.type,
    required this.version,
    required this.payload,
  });

  ChatMessageFrameTypeEnum type;

  ChatMessageFrameVersionEnum version;

  ChatMessage payload;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ChatMessageFrame &&
          other.type == type &&
          other.version == version &&
          other.payload == payload;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (type.hashCode) + (version.hashCode) + (payload.hashCode);

  @override
  String toString() =>
      'ChatMessageFrame[type=$type, version=$version, payload=$payload]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'type'] = this.type;
    json[r'version'] = this.version;
    json[r'payload'] = this.payload;
    return json;
  }

  /// Returns a new [ChatMessageFrame] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ChatMessageFrame? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'type'),
            'Required key "ChatMessageFrame[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "ChatMessageFrame[type]" has a null value in JSON.');
        assert(json.containsKey(r'version'),
            'Required key "ChatMessageFrame[version]" is missing from JSON.');
        assert(json[r'version'] != null,
            'Required key "ChatMessageFrame[version]" has a null value in JSON.');
        assert(json.containsKey(r'payload'),
            'Required key "ChatMessageFrame[payload]" is missing from JSON.');
        assert(json[r'payload'] != null,
            'Required key "ChatMessageFrame[payload]" has a null value in JSON.');
        return true;
      }());

      return ChatMessageFrame(
        type: ChatMessageFrameTypeEnum.fromJson(json[r'type'])!,
        version: ChatMessageFrameVersionEnum.fromJson(json[r'version'])!,
        payload: ChatMessage.fromJson(json[r'payload'])!,
      );
    }
    return null;
  }

  static List<ChatMessageFrame> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatMessageFrame>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatMessageFrame.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ChatMessageFrame> mapFromJson(dynamic json) {
    final map = <String, ChatMessageFrame>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ChatMessageFrame.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ChatMessageFrame-objects as value to a dart map
  static Map<String, List<ChatMessageFrame>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ChatMessageFrame>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ChatMessageFrame.listFromJson(
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

enum ChatMessageFrameTypeEnum {
  message._(r'message'),
  messageEdited._(r'message_edited'),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatMessageFrameTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ChatMessageFrameTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatMessageFrameTypeEnum? fromJson(dynamic value) =>
      ChatMessageFrameTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatMessageFrameTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatMessageFrameTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatMessageFrameTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatMessageFrameTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatMessageFrameTypeEnum] to String,
/// and [decode] dynamic data back to [ChatMessageFrameTypeEnum].
class ChatMessageFrameTypeEnumTypeTransformer {
  factory ChatMessageFrameTypeEnumTypeTransformer() =>
      _instance ??= const ChatMessageFrameTypeEnumTypeTransformer._();

  const ChatMessageFrameTypeEnumTypeTransformer._();

  String encode(ChatMessageFrameTypeEnum data) => data._value;

  /// Returns the instance of [ChatMessageFrameTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatMessageFrameTypeEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ChatMessageFrameTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'message':
          return ChatMessageFrameTypeEnum.message;
        case r'message_edited':
          return ChatMessageFrameTypeEnum.messageEdited;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatMessageFrameTypeEnumTypeTransformer? _instance;
}

enum ChatMessageFrameVersionEnum {
  number1._(1),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatMessageFrameVersionEnum._(this._value);

  /// The underlying value of this enum member.
  final int _value;

  @override
  String toString() => _value.toString();

  /// Encodes this enum as a value suitable for JSON.
  int toJson() => _value;

  /// Returns the instance of [ChatMessageFrameVersionEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatMessageFrameVersionEnum? fromJson(dynamic value) =>
      ChatMessageFrameVersionEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatMessageFrameVersionEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatMessageFrameVersionEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatMessageFrameVersionEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatMessageFrameVersionEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatMessageFrameVersionEnum] to int,
/// and [decode] dynamic data back to [ChatMessageFrameVersionEnum].
class ChatMessageFrameVersionEnumTypeTransformer {
  factory ChatMessageFrameVersionEnumTypeTransformer() =>
      _instance ??= const ChatMessageFrameVersionEnumTypeTransformer._();

  const ChatMessageFrameVersionEnumTypeTransformer._();

  int encode(ChatMessageFrameVersionEnum data) => data._value;

  /// Returns the instance of [ChatMessageFrameVersionEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatMessageFrameVersionEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ChatMessageFrameVersionEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case 1:
          return ChatMessageFrameVersionEnum.number1;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatMessageFrameVersionEnumTypeTransformer? _instance;
}
