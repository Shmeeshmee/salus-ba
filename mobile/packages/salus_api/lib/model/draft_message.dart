//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DraftMessage {
  /// Returns a new [DraftMessage] instance.
  DraftMessage({
    required this.type,
    required this.version,
    required this.payload,
  });

  DraftMessageTypeEnum type;

  DraftMessageVersionEnum version;

  DraftEvent payload;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DraftMessage &&
          other.type == type &&
          other.version == version &&
          other.payload == payload;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (type.hashCode) + (version.hashCode) + (payload.hashCode);

  @override
  String toString() =>
      'DraftMessage[type=$type, version=$version, payload=$payload]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'type'] = this.type;
    json[r'version'] = this.version;
    json[r'payload'] = this.payload;
    return json;
  }

  /// Returns a new [DraftMessage] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static DraftMessage? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'type'),
            'Required key "DraftMessage[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "DraftMessage[type]" has a null value in JSON.');
        assert(json.containsKey(r'version'),
            'Required key "DraftMessage[version]" is missing from JSON.');
        assert(json[r'version'] != null,
            'Required key "DraftMessage[version]" has a null value in JSON.');
        assert(json.containsKey(r'payload'),
            'Required key "DraftMessage[payload]" is missing from JSON.');
        assert(json[r'payload'] != null,
            'Required key "DraftMessage[payload]" has a null value in JSON.');
        return true;
      }());

      return DraftMessage(
        type: DraftMessageTypeEnum.fromJson(json[r'type'])!,
        version: DraftMessageVersionEnum.fromJson(json[r'version'])!,
        payload: DraftEvent.fromJson(json[r'payload'])!,
      );
    }
    return null;
  }

  static List<DraftMessage> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DraftMessage>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DraftMessage.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, DraftMessage> mapFromJson(dynamic json) {
    final map = <String, DraftMessage>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = DraftMessage.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of DraftMessage-objects as value to a dart map
  static Map<String, List<DraftMessage>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<DraftMessage>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = DraftMessage.listFromJson(
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

enum DraftMessageTypeEnum {
  draft._(r'draft'),
  ;

  /// Instantiate a new enum with the provided value.
  const DraftMessageTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [DraftMessageTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static DraftMessageTypeEnum? fromJson(dynamic value) =>
      DraftMessageTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [DraftMessageTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<DraftMessageTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DraftMessageTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DraftMessageTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [DraftMessageTypeEnum] to String,
/// and [decode] dynamic data back to [DraftMessageTypeEnum].
class DraftMessageTypeEnumTypeTransformer {
  factory DraftMessageTypeEnumTypeTransformer() =>
      _instance ??= const DraftMessageTypeEnumTypeTransformer._();

  const DraftMessageTypeEnumTypeTransformer._();

  String encode(DraftMessageTypeEnum data) => data._value;

  /// Returns the instance of [DraftMessageTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  DraftMessageTypeEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is DraftMessageTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'draft':
          return DraftMessageTypeEnum.draft;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static DraftMessageTypeEnumTypeTransformer? _instance;
}

enum DraftMessageVersionEnum {
  number1._(1),
  ;

  /// Instantiate a new enum with the provided value.
  const DraftMessageVersionEnum._(this._value);

  /// The underlying value of this enum member.
  final int _value;

  @override
  String toString() => _value.toString();

  /// Encodes this enum as a value suitable for JSON.
  int toJson() => _value;

  /// Returns the instance of [DraftMessageVersionEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static DraftMessageVersionEnum? fromJson(dynamic value) =>
      DraftMessageVersionEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [DraftMessageVersionEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<DraftMessageVersionEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DraftMessageVersionEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DraftMessageVersionEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [DraftMessageVersionEnum] to int,
/// and [decode] dynamic data back to [DraftMessageVersionEnum].
class DraftMessageVersionEnumTypeTransformer {
  factory DraftMessageVersionEnumTypeTransformer() =>
      _instance ??= const DraftMessageVersionEnumTypeTransformer._();

  const DraftMessageVersionEnumTypeTransformer._();

  int encode(DraftMessageVersionEnum data) => data._value;

  /// Returns the instance of [DraftMessageVersionEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  DraftMessageVersionEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is DraftMessageVersionEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case 1:
          return DraftMessageVersionEnum.number1;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static DraftMessageVersionEnumTypeTransformer? _instance;
}
