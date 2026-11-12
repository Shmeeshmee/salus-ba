//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ChatReadCommand {
  /// Returns a new [ChatReadCommand] instance.
  ChatReadCommand({
    required this.type,
    required this.version,
    required this.payload,
  });

  ChatReadCommandTypeEnum type;

  ChatReadCommandVersionEnum version;

  ReadPosition payload;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ChatReadCommand &&
          other.type == type &&
          other.version == version &&
          other.payload == payload;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (type.hashCode) + (version.hashCode) + (payload.hashCode);

  @override
  String toString() =>
      'ChatReadCommand[type=$type, version=$version, payload=$payload]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'type'] = this.type;
    json[r'version'] = this.version;
    json[r'payload'] = this.payload;
    return json;
  }

  /// Returns a new [ChatReadCommand] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ChatReadCommand? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'type'),
            'Required key "ChatReadCommand[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "ChatReadCommand[type]" has a null value in JSON.');
        assert(json.containsKey(r'version'),
            'Required key "ChatReadCommand[version]" is missing from JSON.');
        assert(json[r'version'] != null,
            'Required key "ChatReadCommand[version]" has a null value in JSON.');
        assert(json.containsKey(r'payload'),
            'Required key "ChatReadCommand[payload]" is missing from JSON.');
        assert(json[r'payload'] != null,
            'Required key "ChatReadCommand[payload]" has a null value in JSON.');
        return true;
      }());

      return ChatReadCommand(
        type: ChatReadCommandTypeEnum.fromJson(json[r'type'])!,
        version: ChatReadCommandVersionEnum.fromJson(json[r'version'])!,
        payload: ReadPosition.fromJson(json[r'payload'])!,
      );
    }
    return null;
  }

  static List<ChatReadCommand> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatReadCommand>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatReadCommand.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ChatReadCommand> mapFromJson(dynamic json) {
    final map = <String, ChatReadCommand>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ChatReadCommand.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ChatReadCommand-objects as value to a dart map
  static Map<String, List<ChatReadCommand>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ChatReadCommand>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ChatReadCommand.listFromJson(
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

enum ChatReadCommandTypeEnum {
  read._(r'read'),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatReadCommandTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ChatReadCommandTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatReadCommandTypeEnum? fromJson(dynamic value) =>
      ChatReadCommandTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatReadCommandTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatReadCommandTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatReadCommandTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatReadCommandTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatReadCommandTypeEnum] to String,
/// and [decode] dynamic data back to [ChatReadCommandTypeEnum].
class ChatReadCommandTypeEnumTypeTransformer {
  factory ChatReadCommandTypeEnumTypeTransformer() =>
      _instance ??= const ChatReadCommandTypeEnumTypeTransformer._();

  const ChatReadCommandTypeEnumTypeTransformer._();

  String encode(ChatReadCommandTypeEnum data) => data._value;

  /// Returns the instance of [ChatReadCommandTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatReadCommandTypeEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ChatReadCommandTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'read':
          return ChatReadCommandTypeEnum.read;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatReadCommandTypeEnumTypeTransformer? _instance;
}

enum ChatReadCommandVersionEnum {
  number1._(1),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatReadCommandVersionEnum._(this._value);

  /// The underlying value of this enum member.
  final int _value;

  @override
  String toString() => _value.toString();

  /// Encodes this enum as a value suitable for JSON.
  int toJson() => _value;

  /// Returns the instance of [ChatReadCommandVersionEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatReadCommandVersionEnum? fromJson(dynamic value) =>
      ChatReadCommandVersionEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatReadCommandVersionEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatReadCommandVersionEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatReadCommandVersionEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatReadCommandVersionEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatReadCommandVersionEnum] to int,
/// and [decode] dynamic data back to [ChatReadCommandVersionEnum].
class ChatReadCommandVersionEnumTypeTransformer {
  factory ChatReadCommandVersionEnumTypeTransformer() =>
      _instance ??= const ChatReadCommandVersionEnumTypeTransformer._();

  const ChatReadCommandVersionEnumTypeTransformer._();

  int encode(ChatReadCommandVersionEnum data) => data._value;

  /// Returns the instance of [ChatReadCommandVersionEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatReadCommandVersionEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ChatReadCommandVersionEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case 1:
          return ChatReadCommandVersionEnum.number1;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatReadCommandVersionEnumTypeTransformer? _instance;
}
