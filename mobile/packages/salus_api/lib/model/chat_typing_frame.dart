//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ChatTypingFrame {
  /// Returns a new [ChatTypingFrame] instance.
  ChatTypingFrame({
    required this.type,
    required this.version,
    required this.payload,
  });

  ChatTypingFrameTypeEnum type;

  ChatTypingFrameVersionEnum version;

  ChatTyping payload;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ChatTypingFrame &&
          other.type == type &&
          other.version == version &&
          other.payload == payload;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (type.hashCode) + (version.hashCode) + (payload.hashCode);

  @override
  String toString() =>
      'ChatTypingFrame[type=$type, version=$version, payload=$payload]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'type'] = this.type;
    json[r'version'] = this.version;
    json[r'payload'] = this.payload;
    return json;
  }

  /// Returns a new [ChatTypingFrame] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ChatTypingFrame? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'type'),
            'Required key "ChatTypingFrame[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "ChatTypingFrame[type]" has a null value in JSON.');
        assert(json.containsKey(r'version'),
            'Required key "ChatTypingFrame[version]" is missing from JSON.');
        assert(json[r'version'] != null,
            'Required key "ChatTypingFrame[version]" has a null value in JSON.');
        assert(json.containsKey(r'payload'),
            'Required key "ChatTypingFrame[payload]" is missing from JSON.');
        assert(json[r'payload'] != null,
            'Required key "ChatTypingFrame[payload]" has a null value in JSON.');
        return true;
      }());

      return ChatTypingFrame(
        type: ChatTypingFrameTypeEnum.fromJson(json[r'type'])!,
        version: ChatTypingFrameVersionEnum.fromJson(json[r'version'])!,
        payload: ChatTyping.fromJson(json[r'payload'])!,
      );
    }
    return null;
  }

  static List<ChatTypingFrame> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatTypingFrame>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatTypingFrame.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ChatTypingFrame> mapFromJson(dynamic json) {
    final map = <String, ChatTypingFrame>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ChatTypingFrame.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ChatTypingFrame-objects as value to a dart map
  static Map<String, List<ChatTypingFrame>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ChatTypingFrame>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ChatTypingFrame.listFromJson(
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

enum ChatTypingFrameTypeEnum {
  typing._(r'typing'),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatTypingFrameTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ChatTypingFrameTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatTypingFrameTypeEnum? fromJson(dynamic value) =>
      ChatTypingFrameTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatTypingFrameTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatTypingFrameTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatTypingFrameTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatTypingFrameTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatTypingFrameTypeEnum] to String,
/// and [decode] dynamic data back to [ChatTypingFrameTypeEnum].
class ChatTypingFrameTypeEnumTypeTransformer {
  factory ChatTypingFrameTypeEnumTypeTransformer() =>
      _instance ??= const ChatTypingFrameTypeEnumTypeTransformer._();

  const ChatTypingFrameTypeEnumTypeTransformer._();

  String encode(ChatTypingFrameTypeEnum data) => data._value;

  /// Returns the instance of [ChatTypingFrameTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatTypingFrameTypeEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ChatTypingFrameTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'typing':
          return ChatTypingFrameTypeEnum.typing;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatTypingFrameTypeEnumTypeTransformer? _instance;
}

enum ChatTypingFrameVersionEnum {
  number1._(1),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatTypingFrameVersionEnum._(this._value);

  /// The underlying value of this enum member.
  final int _value;

  @override
  String toString() => _value.toString();

  /// Encodes this enum as a value suitable for JSON.
  int toJson() => _value;

  /// Returns the instance of [ChatTypingFrameVersionEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatTypingFrameVersionEnum? fromJson(dynamic value) =>
      ChatTypingFrameVersionEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatTypingFrameVersionEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatTypingFrameVersionEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatTypingFrameVersionEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatTypingFrameVersionEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatTypingFrameVersionEnum] to int,
/// and [decode] dynamic data back to [ChatTypingFrameVersionEnum].
class ChatTypingFrameVersionEnumTypeTransformer {
  factory ChatTypingFrameVersionEnumTypeTransformer() =>
      _instance ??= const ChatTypingFrameVersionEnumTypeTransformer._();

  const ChatTypingFrameVersionEnumTypeTransformer._();

  int encode(ChatTypingFrameVersionEnum data) => data._value;

  /// Returns the instance of [ChatTypingFrameVersionEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatTypingFrameVersionEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ChatTypingFrameVersionEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case 1:
          return ChatTypingFrameVersionEnum.number1;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatTypingFrameVersionEnumTypeTransformer? _instance;
}
