//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class TranscriptMessage {
  /// Returns a new [TranscriptMessage] instance.
  TranscriptMessage({
    required this.type,
    required this.version,
    required this.payload,
  });

  TranscriptMessageTypeEnum type;

  TranscriptMessageVersionEnum version;

  TranscriptEvent payload;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is TranscriptMessage &&
          other.type == type &&
          other.version == version &&
          other.payload == payload;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (type.hashCode) + (version.hashCode) + (payload.hashCode);

  @override
  String toString() =>
      'TranscriptMessage[type=$type, version=$version, payload=$payload]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'type'] = this.type;
    json[r'version'] = this.version;
    json[r'payload'] = this.payload;
    return json;
  }

  /// Returns a new [TranscriptMessage] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static TranscriptMessage? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'type'),
            'Required key "TranscriptMessage[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "TranscriptMessage[type]" has a null value in JSON.');
        assert(json.containsKey(r'version'),
            'Required key "TranscriptMessage[version]" is missing from JSON.');
        assert(json[r'version'] != null,
            'Required key "TranscriptMessage[version]" has a null value in JSON.');
        assert(json.containsKey(r'payload'),
            'Required key "TranscriptMessage[payload]" is missing from JSON.');
        assert(json[r'payload'] != null,
            'Required key "TranscriptMessage[payload]" has a null value in JSON.');
        return true;
      }());

      return TranscriptMessage(
        type: TranscriptMessageTypeEnum.fromJson(json[r'type'])!,
        version: TranscriptMessageVersionEnum.fromJson(json[r'version'])!,
        payload: TranscriptEvent.fromJson(json[r'payload'])!,
      );
    }
    return null;
  }

  static List<TranscriptMessage> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <TranscriptMessage>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = TranscriptMessage.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, TranscriptMessage> mapFromJson(dynamic json) {
    final map = <String, TranscriptMessage>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = TranscriptMessage.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of TranscriptMessage-objects as value to a dart map
  static Map<String, List<TranscriptMessage>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<TranscriptMessage>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = TranscriptMessage.listFromJson(
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

enum TranscriptMessageTypeEnum {
  transcript._(r'transcript'),
  ;

  /// Instantiate a new enum with the provided value.
  const TranscriptMessageTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [TranscriptMessageTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static TranscriptMessageTypeEnum? fromJson(dynamic value) =>
      TranscriptMessageTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [TranscriptMessageTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<TranscriptMessageTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <TranscriptMessageTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = TranscriptMessageTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [TranscriptMessageTypeEnum] to String,
/// and [decode] dynamic data back to [TranscriptMessageTypeEnum].
class TranscriptMessageTypeEnumTypeTransformer {
  factory TranscriptMessageTypeEnumTypeTransformer() =>
      _instance ??= const TranscriptMessageTypeEnumTypeTransformer._();

  const TranscriptMessageTypeEnumTypeTransformer._();

  String encode(TranscriptMessageTypeEnum data) => data._value;

  /// Returns the instance of [TranscriptMessageTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  TranscriptMessageTypeEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is TranscriptMessageTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'transcript':
          return TranscriptMessageTypeEnum.transcript;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static TranscriptMessageTypeEnumTypeTransformer? _instance;
}

enum TranscriptMessageVersionEnum {
  number1._(1),
  ;

  /// Instantiate a new enum with the provided value.
  const TranscriptMessageVersionEnum._(this._value);

  /// The underlying value of this enum member.
  final int _value;

  @override
  String toString() => _value.toString();

  /// Encodes this enum as a value suitable for JSON.
  int toJson() => _value;

  /// Returns the instance of [TranscriptMessageVersionEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static TranscriptMessageVersionEnum? fromJson(dynamic value) =>
      TranscriptMessageVersionEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [TranscriptMessageVersionEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<TranscriptMessageVersionEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <TranscriptMessageVersionEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = TranscriptMessageVersionEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [TranscriptMessageVersionEnum] to int,
/// and [decode] dynamic data back to [TranscriptMessageVersionEnum].
class TranscriptMessageVersionEnumTypeTransformer {
  factory TranscriptMessageVersionEnumTypeTransformer() =>
      _instance ??= const TranscriptMessageVersionEnumTypeTransformer._();

  const TranscriptMessageVersionEnumTypeTransformer._();

  int encode(TranscriptMessageVersionEnum data) => data._value;

  /// Returns the instance of [TranscriptMessageVersionEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  TranscriptMessageVersionEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is TranscriptMessageVersionEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case 1:
          return TranscriptMessageVersionEnum.number1;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static TranscriptMessageVersionEnumTypeTransformer? _instance;
}
