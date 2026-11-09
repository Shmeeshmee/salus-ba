//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SuggestionStatusMessage {
  /// Returns a new [SuggestionStatusMessage] instance.
  SuggestionStatusMessage({
    required this.type,
    required this.version,
    required this.payload,
  });

  SuggestionStatusMessageTypeEnum type;

  SuggestionStatusMessageVersionEnum version;

  SuggestionStatus payload;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SuggestionStatusMessage &&
          other.type == type &&
          other.version == version &&
          other.payload == payload;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (type.hashCode) + (version.hashCode) + (payload.hashCode);

  @override
  String toString() =>
      'SuggestionStatusMessage[type=$type, version=$version, payload=$payload]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'type'] = this.type;
    json[r'version'] = this.version;
    json[r'payload'] = this.payload;
    return json;
  }

  /// Returns a new [SuggestionStatusMessage] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SuggestionStatusMessage? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'type'),
            'Required key "SuggestionStatusMessage[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "SuggestionStatusMessage[type]" has a null value in JSON.');
        assert(json.containsKey(r'version'),
            'Required key "SuggestionStatusMessage[version]" is missing from JSON.');
        assert(json[r'version'] != null,
            'Required key "SuggestionStatusMessage[version]" has a null value in JSON.');
        assert(json.containsKey(r'payload'),
            'Required key "SuggestionStatusMessage[payload]" is missing from JSON.');
        assert(json[r'payload'] != null,
            'Required key "SuggestionStatusMessage[payload]" has a null value in JSON.');
        return true;
      }());

      return SuggestionStatusMessage(
        type: SuggestionStatusMessageTypeEnum.fromJson(json[r'type'])!,
        version: SuggestionStatusMessageVersionEnum.fromJson(json[r'version'])!,
        payload: SuggestionStatus.fromJson(json[r'payload'])!,
      );
    }
    return null;
  }

  static List<SuggestionStatusMessage> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SuggestionStatusMessage>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SuggestionStatusMessage.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SuggestionStatusMessage> mapFromJson(dynamic json) {
    final map = <String, SuggestionStatusMessage>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SuggestionStatusMessage.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SuggestionStatusMessage-objects as value to a dart map
  static Map<String, List<SuggestionStatusMessage>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SuggestionStatusMessage>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SuggestionStatusMessage.listFromJson(
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

enum SuggestionStatusMessageTypeEnum {
  suggestionStatus._(r'suggestion_status'),
  ;

  /// Instantiate a new enum with the provided value.
  const SuggestionStatusMessageTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [SuggestionStatusMessageTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static SuggestionStatusMessageTypeEnum? fromJson(dynamic value) =>
      SuggestionStatusMessageTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [SuggestionStatusMessageTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<SuggestionStatusMessageTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SuggestionStatusMessageTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SuggestionStatusMessageTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [SuggestionStatusMessageTypeEnum] to String,
/// and [decode] dynamic data back to [SuggestionStatusMessageTypeEnum].
class SuggestionStatusMessageTypeEnumTypeTransformer {
  factory SuggestionStatusMessageTypeEnumTypeTransformer() =>
      _instance ??= const SuggestionStatusMessageTypeEnumTypeTransformer._();

  const SuggestionStatusMessageTypeEnumTypeTransformer._();

  String encode(SuggestionStatusMessageTypeEnum data) => data._value;

  /// Returns the instance of [SuggestionStatusMessageTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  SuggestionStatusMessageTypeEnum? decode(dynamic data,
      {bool allowNull = true}) {
    if (data is SuggestionStatusMessageTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'suggestion_status':
          return SuggestionStatusMessageTypeEnum.suggestionStatus;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static SuggestionStatusMessageTypeEnumTypeTransformer? _instance;
}

enum SuggestionStatusMessageVersionEnum {
  number1._(1),
  ;

  /// Instantiate a new enum with the provided value.
  const SuggestionStatusMessageVersionEnum._(this._value);

  /// The underlying value of this enum member.
  final int _value;

  @override
  String toString() => _value.toString();

  /// Encodes this enum as a value suitable for JSON.
  int toJson() => _value;

  /// Returns the instance of [SuggestionStatusMessageVersionEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static SuggestionStatusMessageVersionEnum? fromJson(dynamic value) =>
      SuggestionStatusMessageVersionEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [SuggestionStatusMessageVersionEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<SuggestionStatusMessageVersionEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SuggestionStatusMessageVersionEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SuggestionStatusMessageVersionEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [SuggestionStatusMessageVersionEnum] to int,
/// and [decode] dynamic data back to [SuggestionStatusMessageVersionEnum].
class SuggestionStatusMessageVersionEnumTypeTransformer {
  factory SuggestionStatusMessageVersionEnumTypeTransformer() =>
      _instance ??= const SuggestionStatusMessageVersionEnumTypeTransformer._();

  const SuggestionStatusMessageVersionEnumTypeTransformer._();

  int encode(SuggestionStatusMessageVersionEnum data) => data._value;

  /// Returns the instance of [SuggestionStatusMessageVersionEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  SuggestionStatusMessageVersionEnum? decode(dynamic data,
      {bool allowNull = true}) {
    if (data is SuggestionStatusMessageVersionEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case 1:
          return SuggestionStatusMessageVersionEnum.number1;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static SuggestionStatusMessageVersionEnumTypeTransformer? _instance;
}
