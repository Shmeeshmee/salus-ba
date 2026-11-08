//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SuggestionMessage {
  /// Returns a new [SuggestionMessage] instance.
  SuggestionMessage({
    required this.type,
    required this.version,
    required this.payload,
  });

  SuggestionMessageTypeEnum type;

  SuggestionMessageVersionEnum version;

  SuggestionEvent payload;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SuggestionMessage &&
          other.type == type &&
          other.version == version &&
          other.payload == payload;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (type.hashCode) + (version.hashCode) + (payload.hashCode);

  @override
  String toString() =>
      'SuggestionMessage[type=$type, version=$version, payload=$payload]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'type'] = this.type;
    json[r'version'] = this.version;
    json[r'payload'] = this.payload;
    return json;
  }

  /// Returns a new [SuggestionMessage] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SuggestionMessage? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'type'),
            'Required key "SuggestionMessage[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "SuggestionMessage[type]" has a null value in JSON.');
        assert(json.containsKey(r'version'),
            'Required key "SuggestionMessage[version]" is missing from JSON.');
        assert(json[r'version'] != null,
            'Required key "SuggestionMessage[version]" has a null value in JSON.');
        assert(json.containsKey(r'payload'),
            'Required key "SuggestionMessage[payload]" is missing from JSON.');
        assert(json[r'payload'] != null,
            'Required key "SuggestionMessage[payload]" has a null value in JSON.');
        return true;
      }());

      return SuggestionMessage(
        type: SuggestionMessageTypeEnum.fromJson(json[r'type'])!,
        version: SuggestionMessageVersionEnum.fromJson(json[r'version'])!,
        payload: SuggestionEvent.fromJson(json[r'payload'])!,
      );
    }
    return null;
  }

  static List<SuggestionMessage> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SuggestionMessage>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SuggestionMessage.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SuggestionMessage> mapFromJson(dynamic json) {
    final map = <String, SuggestionMessage>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SuggestionMessage.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SuggestionMessage-objects as value to a dart map
  static Map<String, List<SuggestionMessage>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SuggestionMessage>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SuggestionMessage.listFromJson(
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

enum SuggestionMessageTypeEnum {
  suggestion._(r'suggestion'),
  ;

  /// Instantiate a new enum with the provided value.
  const SuggestionMessageTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [SuggestionMessageTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static SuggestionMessageTypeEnum? fromJson(dynamic value) =>
      SuggestionMessageTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [SuggestionMessageTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<SuggestionMessageTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SuggestionMessageTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SuggestionMessageTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [SuggestionMessageTypeEnum] to String,
/// and [decode] dynamic data back to [SuggestionMessageTypeEnum].
class SuggestionMessageTypeEnumTypeTransformer {
  factory SuggestionMessageTypeEnumTypeTransformer() =>
      _instance ??= const SuggestionMessageTypeEnumTypeTransformer._();

  const SuggestionMessageTypeEnumTypeTransformer._();

  String encode(SuggestionMessageTypeEnum data) => data._value;

  /// Returns the instance of [SuggestionMessageTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  SuggestionMessageTypeEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is SuggestionMessageTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'suggestion':
          return SuggestionMessageTypeEnum.suggestion;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static SuggestionMessageTypeEnumTypeTransformer? _instance;
}

enum SuggestionMessageVersionEnum {
  number1._(1),
  ;

  /// Instantiate a new enum with the provided value.
  const SuggestionMessageVersionEnum._(this._value);

  /// The underlying value of this enum member.
  final int _value;

  @override
  String toString() => _value.toString();

  /// Encodes this enum as a value suitable for JSON.
  int toJson() => _value;

  /// Returns the instance of [SuggestionMessageVersionEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static SuggestionMessageVersionEnum? fromJson(dynamic value) =>
      SuggestionMessageVersionEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [SuggestionMessageVersionEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<SuggestionMessageVersionEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SuggestionMessageVersionEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SuggestionMessageVersionEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [SuggestionMessageVersionEnum] to int,
/// and [decode] dynamic data back to [SuggestionMessageVersionEnum].
class SuggestionMessageVersionEnumTypeTransformer {
  factory SuggestionMessageVersionEnumTypeTransformer() =>
      _instance ??= const SuggestionMessageVersionEnumTypeTransformer._();

  const SuggestionMessageVersionEnumTypeTransformer._();

  int encode(SuggestionMessageVersionEnum data) => data._value;

  /// Returns the instance of [SuggestionMessageVersionEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  SuggestionMessageVersionEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is SuggestionMessageVersionEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case 1:
          return SuggestionMessageVersionEnum.number1;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static SuggestionMessageVersionEnumTypeTransformer? _instance;
}
