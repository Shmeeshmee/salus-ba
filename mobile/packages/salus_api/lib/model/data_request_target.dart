//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

/// `account`: the patient's Salus account, answered by Salus; `clinic`: their record at one clinic, answered by its admin.
enum DataRequestTarget {
  account._(r'account'),
  clinic._(r'clinic'),
  ;

  /// Instantiate a new enum with the provided value.
  const DataRequestTarget._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [DataRequestTarget] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static DataRequestTarget? fromJson(dynamic value) =>
      DataRequestTargetTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [DataRequestTarget]
  /// that were successfully decoded from the passed [JSON][json].
  static List<DataRequestTarget> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DataRequestTarget>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DataRequestTarget.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [DataRequestTarget] to String,
/// and [decode] dynamic data back to [DataRequestTarget].
class DataRequestTargetTypeTransformer {
  factory DataRequestTargetTypeTransformer() =>
      _instance ??= const DataRequestTargetTypeTransformer._();

  const DataRequestTargetTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(DataRequestTarget data) => data._value;

  /// Returns the instance of [DataRequestTarget] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  DataRequestTarget? decode(dynamic data, {bool allowNull = true}) {
    if (data is DataRequestTarget) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'account':
          return DataRequestTarget.account;
        case r'clinic':
          return DataRequestTarget.clinic;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static DataRequestTargetTypeTransformer? _instance;
}
