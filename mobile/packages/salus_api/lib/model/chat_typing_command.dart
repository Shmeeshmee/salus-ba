//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ChatTypingCommand {
  /// Returns a new [ChatTypingCommand] instance.
  ChatTypingCommand({
    required this.type,
    required this.version,
    required this.payload,
  });

  ChatTypingCommandTypeEnum type;

  ChatTypingCommandVersionEnum version;

  ChatTyping payload;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ChatTypingCommand &&
          other.type == type &&
          other.version == version &&
          other.payload == payload;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (type.hashCode) + (version.hashCode) + (payload.hashCode);

  @override
  String toString() =>
      'ChatTypingCommand[type=$type, version=$version, payload=$payload]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'type'] = this.type;
    json[r'version'] = this.version;
    json[r'payload'] = this.payload;
    return json;
  }

  /// Returns a new [ChatTypingCommand] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ChatTypingCommand? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'type'),
            'Required key "ChatTypingCommand[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "ChatTypingCommand[type]" has a null value in JSON.');
        assert(json.containsKey(r'version'),
            'Required key "ChatTypingCommand[version]" is missing from JSON.');
        assert(json[r'version'] != null,
            'Required key "ChatTypingCommand[version]" has a null value in JSON.');
        assert(json.containsKey(r'payload'),
            'Required key "ChatTypingCommand[payload]" is missing from JSON.');
        assert(json[r'payload'] != null,
            'Required key "ChatTypingCommand[payload]" has a null value in JSON.');
        return true;
      }());

      return ChatTypingCommand(
        type: ChatTypingCommandTypeEnum.fromJson(json[r'type'])!,
        version: ChatTypingCommandVersionEnum.fromJson(json[r'version'])!,
        payload: ChatTyping.fromJson(json[r'payload'])!,
      );
    }
    return null;
  }

  static List<ChatTypingCommand> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatTypingCommand>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatTypingCommand.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ChatTypingCommand> mapFromJson(dynamic json) {
    final map = <String, ChatTypingCommand>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ChatTypingCommand.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ChatTypingCommand-objects as value to a dart map
  static Map<String, List<ChatTypingCommand>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ChatTypingCommand>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ChatTypingCommand.listFromJson(
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

enum ChatTypingCommandTypeEnum {
  typing._(r'typing'),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatTypingCommandTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ChatTypingCommandTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatTypingCommandTypeEnum? fromJson(dynamic value) =>
      ChatTypingCommandTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatTypingCommandTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatTypingCommandTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatTypingCommandTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatTypingCommandTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatTypingCommandTypeEnum] to String,
/// and [decode] dynamic data back to [ChatTypingCommandTypeEnum].
class ChatTypingCommandTypeEnumTypeTransformer {
  factory ChatTypingCommandTypeEnumTypeTransformer() =>
      _instance ??= const ChatTypingCommandTypeEnumTypeTransformer._();

  const ChatTypingCommandTypeEnumTypeTransformer._();

  String encode(ChatTypingCommandTypeEnum data) => data._value;

  /// Returns the instance of [ChatTypingCommandTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatTypingCommandTypeEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ChatTypingCommandTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'typing':
          return ChatTypingCommandTypeEnum.typing;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatTypingCommandTypeEnumTypeTransformer? _instance;
}

enum ChatTypingCommandVersionEnum {
  number1._(1),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatTypingCommandVersionEnum._(this._value);

  /// The underlying value of this enum member.
  final int _value;

  @override
  String toString() => _value.toString();

  /// Encodes this enum as a value suitable for JSON.
  int toJson() => _value;

  /// Returns the instance of [ChatTypingCommandVersionEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatTypingCommandVersionEnum? fromJson(dynamic value) =>
      ChatTypingCommandVersionEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatTypingCommandVersionEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatTypingCommandVersionEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatTypingCommandVersionEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatTypingCommandVersionEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatTypingCommandVersionEnum] to int,
/// and [decode] dynamic data back to [ChatTypingCommandVersionEnum].
class ChatTypingCommandVersionEnumTypeTransformer {
  factory ChatTypingCommandVersionEnumTypeTransformer() =>
      _instance ??= const ChatTypingCommandVersionEnumTypeTransformer._();

  const ChatTypingCommandVersionEnumTypeTransformer._();

  int encode(ChatTypingCommandVersionEnum data) => data._value;

  /// Returns the instance of [ChatTypingCommandVersionEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatTypingCommandVersionEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ChatTypingCommandVersionEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case 1:
          return ChatTypingCommandVersionEnum.number1;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatTypingCommandVersionEnumTypeTransformer? _instance;
}
