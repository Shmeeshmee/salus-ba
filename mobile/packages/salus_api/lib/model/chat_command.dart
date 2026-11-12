//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ChatCommand {
  /// Returns a new [ChatCommand] instance.
  ChatCommand({
    required this.type,
    required this.version,
    required this.payload,
  });

  ChatCommandTypeEnum type;

  ChatCommandVersionEnum version;

  SendMessageRequest payload;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ChatCommand &&
          other.type == type &&
          other.version == version &&
          other.payload == payload;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (type.hashCode) + (version.hashCode) + (payload.hashCode);

  @override
  String toString() =>
      'ChatCommand[type=$type, version=$version, payload=$payload]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'type'] = this.type;
    json[r'version'] = this.version;
    json[r'payload'] = this.payload;
    return json;
  }

  /// Returns a new [ChatCommand] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ChatCommand? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'type'),
            'Required key "ChatCommand[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "ChatCommand[type]" has a null value in JSON.');
        assert(json.containsKey(r'version'),
            'Required key "ChatCommand[version]" is missing from JSON.');
        assert(json[r'version'] != null,
            'Required key "ChatCommand[version]" has a null value in JSON.');
        assert(json.containsKey(r'payload'),
            'Required key "ChatCommand[payload]" is missing from JSON.');
        assert(json[r'payload'] != null,
            'Required key "ChatCommand[payload]" has a null value in JSON.');
        return true;
      }());

      return ChatCommand(
        type: ChatCommandTypeEnum.fromJson(json[r'type'])!,
        version: ChatCommandVersionEnum.fromJson(json[r'version'])!,
        payload: SendMessageRequest.fromJson(json[r'payload'])!,
      );
    }
    return null;
  }

  static List<ChatCommand> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatCommand>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatCommand.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ChatCommand> mapFromJson(dynamic json) {
    final map = <String, ChatCommand>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ChatCommand.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ChatCommand-objects as value to a dart map
  static Map<String, List<ChatCommand>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ChatCommand>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ChatCommand.listFromJson(
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

enum ChatCommandTypeEnum {
  send._(r'send'),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatCommandTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ChatCommandTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatCommandTypeEnum? fromJson(dynamic value) =>
      ChatCommandTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatCommandTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatCommandTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatCommandTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatCommandTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatCommandTypeEnum] to String,
/// and [decode] dynamic data back to [ChatCommandTypeEnum].
class ChatCommandTypeEnumTypeTransformer {
  factory ChatCommandTypeEnumTypeTransformer() =>
      _instance ??= const ChatCommandTypeEnumTypeTransformer._();

  const ChatCommandTypeEnumTypeTransformer._();

  String encode(ChatCommandTypeEnum data) => data._value;

  /// Returns the instance of [ChatCommandTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatCommandTypeEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ChatCommandTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'send':
          return ChatCommandTypeEnum.send;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatCommandTypeEnumTypeTransformer? _instance;
}

enum ChatCommandVersionEnum {
  number1._(1),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatCommandVersionEnum._(this._value);

  /// The underlying value of this enum member.
  final int _value;

  @override
  String toString() => _value.toString();

  /// Encodes this enum as a value suitable for JSON.
  int toJson() => _value;

  /// Returns the instance of [ChatCommandVersionEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatCommandVersionEnum? fromJson(dynamic value) =>
      ChatCommandVersionEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatCommandVersionEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatCommandVersionEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatCommandVersionEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatCommandVersionEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatCommandVersionEnum] to int,
/// and [decode] dynamic data back to [ChatCommandVersionEnum].
class ChatCommandVersionEnumTypeTransformer {
  factory ChatCommandVersionEnumTypeTransformer() =>
      _instance ??= const ChatCommandVersionEnumTypeTransformer._();

  const ChatCommandVersionEnumTypeTransformer._();

  int encode(ChatCommandVersionEnum data) => data._value;

  /// Returns the instance of [ChatCommandVersionEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatCommandVersionEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ChatCommandVersionEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case 1:
          return ChatCommandVersionEnum.number1;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatCommandVersionEnumTypeTransformer? _instance;
}
