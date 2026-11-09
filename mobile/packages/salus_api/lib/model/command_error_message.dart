//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class CommandErrorMessage {
  /// Returns a new [CommandErrorMessage] instance.
  CommandErrorMessage({
    required this.type,
    required this.version,
    required this.payload,
  });

  CommandErrorMessageTypeEnum type;

  CommandErrorMessageVersionEnum version;

  CommandError payload;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is CommandErrorMessage &&
          other.type == type &&
          other.version == version &&
          other.payload == payload;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (type.hashCode) + (version.hashCode) + (payload.hashCode);

  @override
  String toString() =>
      'CommandErrorMessage[type=$type, version=$version, payload=$payload]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'type'] = this.type;
    json[r'version'] = this.version;
    json[r'payload'] = this.payload;
    return json;
  }

  /// Returns a new [CommandErrorMessage] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static CommandErrorMessage? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'type'),
            'Required key "CommandErrorMessage[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "CommandErrorMessage[type]" has a null value in JSON.');
        assert(json.containsKey(r'version'),
            'Required key "CommandErrorMessage[version]" is missing from JSON.');
        assert(json[r'version'] != null,
            'Required key "CommandErrorMessage[version]" has a null value in JSON.');
        assert(json.containsKey(r'payload'),
            'Required key "CommandErrorMessage[payload]" is missing from JSON.');
        assert(json[r'payload'] != null,
            'Required key "CommandErrorMessage[payload]" has a null value in JSON.');
        return true;
      }());

      return CommandErrorMessage(
        type: CommandErrorMessageTypeEnum.fromJson(json[r'type'])!,
        version: CommandErrorMessageVersionEnum.fromJson(json[r'version'])!,
        payload: CommandError.fromJson(json[r'payload'])!,
      );
    }
    return null;
  }

  static List<CommandErrorMessage> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <CommandErrorMessage>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = CommandErrorMessage.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, CommandErrorMessage> mapFromJson(dynamic json) {
    final map = <String, CommandErrorMessage>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = CommandErrorMessage.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of CommandErrorMessage-objects as value to a dart map
  static Map<String, List<CommandErrorMessage>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<CommandErrorMessage>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = CommandErrorMessage.listFromJson(
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

enum CommandErrorMessageTypeEnum {
  commandError._(r'command_error'),
  ;

  /// Instantiate a new enum with the provided value.
  const CommandErrorMessageTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [CommandErrorMessageTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static CommandErrorMessageTypeEnum? fromJson(dynamic value) =>
      CommandErrorMessageTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [CommandErrorMessageTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<CommandErrorMessageTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <CommandErrorMessageTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = CommandErrorMessageTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [CommandErrorMessageTypeEnum] to String,
/// and [decode] dynamic data back to [CommandErrorMessageTypeEnum].
class CommandErrorMessageTypeEnumTypeTransformer {
  factory CommandErrorMessageTypeEnumTypeTransformer() =>
      _instance ??= const CommandErrorMessageTypeEnumTypeTransformer._();

  const CommandErrorMessageTypeEnumTypeTransformer._();

  String encode(CommandErrorMessageTypeEnum data) => data._value;

  /// Returns the instance of [CommandErrorMessageTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  CommandErrorMessageTypeEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is CommandErrorMessageTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'command_error':
          return CommandErrorMessageTypeEnum.commandError;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static CommandErrorMessageTypeEnumTypeTransformer? _instance;
}

enum CommandErrorMessageVersionEnum {
  number1._(1),
  ;

  /// Instantiate a new enum with the provided value.
  const CommandErrorMessageVersionEnum._(this._value);

  /// The underlying value of this enum member.
  final int _value;

  @override
  String toString() => _value.toString();

  /// Encodes this enum as a value suitable for JSON.
  int toJson() => _value;

  /// Returns the instance of [CommandErrorMessageVersionEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static CommandErrorMessageVersionEnum? fromJson(dynamic value) =>
      CommandErrorMessageVersionEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [CommandErrorMessageVersionEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<CommandErrorMessageVersionEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <CommandErrorMessageVersionEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = CommandErrorMessageVersionEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [CommandErrorMessageVersionEnum] to int,
/// and [decode] dynamic data back to [CommandErrorMessageVersionEnum].
class CommandErrorMessageVersionEnumTypeTransformer {
  factory CommandErrorMessageVersionEnumTypeTransformer() =>
      _instance ??= const CommandErrorMessageVersionEnumTypeTransformer._();

  const CommandErrorMessageVersionEnumTypeTransformer._();

  int encode(CommandErrorMessageVersionEnum data) => data._value;

  /// Returns the instance of [CommandErrorMessageVersionEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  CommandErrorMessageVersionEnum? decode(dynamic data,
      {bool allowNull = true}) {
    if (data is CommandErrorMessageVersionEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case 1:
          return CommandErrorMessageVersionEnum.number1;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static CommandErrorMessageVersionEnumTypeTransformer? _instance;
}
