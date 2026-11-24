//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

/// What the patient asks for (GDPR Art. 15 to 21). Access and portability are served by the same export.
enum DataRequestKind {
  access._(r'access'),
  portability._(r'portability'),
  rectification._(r'rectification'),
  restriction._(r'restriction'),
  objection._(r'objection'),
  erasure._(r'erasure'),
  ;

  /// Instantiate a new enum with the provided value.
  const DataRequestKind._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [DataRequestKind] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static DataRequestKind? fromJson(dynamic value) =>
      DataRequestKindTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [DataRequestKind]
  /// that were successfully decoded from the passed [JSON][json].
  static List<DataRequestKind> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DataRequestKind>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DataRequestKind.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [DataRequestKind] to String,
/// and [decode] dynamic data back to [DataRequestKind].
class DataRequestKindTypeTransformer {
  factory DataRequestKindTypeTransformer() =>
      _instance ??= const DataRequestKindTypeTransformer._();

  const DataRequestKindTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(DataRequestKind data) => data._value;

  /// Returns the instance of [DataRequestKind] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  DataRequestKind? decode(dynamic data, {bool allowNull = true}) {
    if (data is DataRequestKind) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'access':
          return DataRequestKind.access;
        case r'portability':
          return DataRequestKind.portability;
        case r'rectification':
          return DataRequestKind.rectification;
        case r'restriction':
          return DataRequestKind.restriction;
        case r'objection':
          return DataRequestKind.objection;
        case r'erasure':
          return DataRequestKind.erasure;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static DataRequestKindTypeTransformer? _instance;
}
