//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class CommandError {
  /// Returns a new [CommandError] instance.
  CommandError({
    required this.command,
    this.id,
    required this.error,
  });

  /// The failed message's `type`; empty when it was no envelope.
  String command;

  /// The suggestion the message named, when it named one.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? id;

  /// `invalid_message`: no envelope; `unknown_command`: a type or version the server does not take; `invalid`: a payload or status the contract does not allow; `not_found`: no such suggestion of the encounter, or an encounter the caller may not see; `consent_required`: after a revocation; `forbidden`: not the encounter's practitioner; `internal`: a server error.
  CommandErrorErrorEnum error;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is CommandError &&
          other.command == command &&
          other.id == id &&
          other.error == error;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (command.hashCode) + (id == null ? 0 : id!.hashCode) + (error.hashCode);

  @override
  String toString() => 'CommandError[command=$command, id=$id, error=$error]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'command'] = this.command;
    if (this.id != null) {
      json[r'id'] = this.id;
    } else {
      json[r'id'] = null;
    }
    json[r'error'] = this.error;
    return json;
  }

  /// Returns a new [CommandError] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static CommandError? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'command'),
            'Required key "CommandError[command]" is missing from JSON.');
        assert(json[r'command'] != null,
            'Required key "CommandError[command]" has a null value in JSON.');
        assert(json.containsKey(r'error'),
            'Required key "CommandError[error]" is missing from JSON.');
        assert(json[r'error'] != null,
            'Required key "CommandError[error]" has a null value in JSON.');
        return true;
      }());

      return CommandError(
        command: mapValueOfType<String>(json, r'command')!,
        id: mapValueOfType<String>(json, r'id'),
        error: CommandErrorErrorEnum.fromJson(json[r'error'])!,
      );
    }
    return null;
  }

  static List<CommandError> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <CommandError>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = CommandError.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, CommandError> mapFromJson(dynamic json) {
    final map = <String, CommandError>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = CommandError.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of CommandError-objects as value to a dart map
  static Map<String, List<CommandError>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<CommandError>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = CommandError.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'command',
    'error',
  };
}

/// `invalid_message`: no envelope; `unknown_command`: a type or version the server does not take; `invalid`: a payload or status the contract does not allow; `not_found`: no such suggestion of the encounter, or an encounter the caller may not see; `consent_required`: after a revocation; `forbidden`: not the encounter's practitioner; `internal`: a server error.
enum CommandErrorErrorEnum {
  invalidMessage._(r'invalid_message'),
  unknownCommand._(r'unknown_command'),
  invalid._(r'invalid'),
  notFound._(r'not_found'),
  consentRequired._(r'consent_required'),
  forbidden._(r'forbidden'),
  internal._(r'internal'),
  ;

  /// Instantiate a new enum with the provided value.
  const CommandErrorErrorEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [CommandErrorErrorEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static CommandErrorErrorEnum? fromJson(dynamic value) =>
      CommandErrorErrorEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [CommandErrorErrorEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<CommandErrorErrorEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <CommandErrorErrorEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = CommandErrorErrorEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [CommandErrorErrorEnum] to String,
/// and [decode] dynamic data back to [CommandErrorErrorEnum].
class CommandErrorErrorEnumTypeTransformer {
  factory CommandErrorErrorEnumTypeTransformer() =>
      _instance ??= const CommandErrorErrorEnumTypeTransformer._();

  const CommandErrorErrorEnumTypeTransformer._();

  String encode(CommandErrorErrorEnum data) => data._value;

  /// Returns the instance of [CommandErrorErrorEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  CommandErrorErrorEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is CommandErrorErrorEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'invalid_message':
          return CommandErrorErrorEnum.invalidMessage;
        case r'unknown_command':
          return CommandErrorErrorEnum.unknownCommand;
        case r'invalid':
          return CommandErrorErrorEnum.invalid;
        case r'not_found':
          return CommandErrorErrorEnum.notFound;
        case r'consent_required':
          return CommandErrorErrorEnum.consentRequired;
        case r'forbidden':
          return CommandErrorErrorEnum.forbidden;
        case r'internal':
          return CommandErrorErrorEnum.internal;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static CommandErrorErrorEnumTypeTransformer? _instance;
}
