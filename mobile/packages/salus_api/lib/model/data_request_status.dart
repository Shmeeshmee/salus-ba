//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

enum DataRequestStatus {
  received._(r'received'),
  inProgress._(r'in_progress'),
  fulfilled._(r'fulfilled'),
  refused._(r'refused'),
  withdrawn._(r'withdrawn'),
  ;

  /// Instantiate a new enum with the provided value.
  const DataRequestStatus._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [DataRequestStatus] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static DataRequestStatus? fromJson(dynamic value) =>
      DataRequestStatusTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [DataRequestStatus]
  /// that were successfully decoded from the passed [JSON][json].
  static List<DataRequestStatus> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DataRequestStatus>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DataRequestStatus.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [DataRequestStatus] to String,
/// and [decode] dynamic data back to [DataRequestStatus].
class DataRequestStatusTypeTransformer {
  factory DataRequestStatusTypeTransformer() =>
      _instance ??= const DataRequestStatusTypeTransformer._();

  const DataRequestStatusTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(DataRequestStatus data) => data._value;

  /// Returns the instance of [DataRequestStatus] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  DataRequestStatus? decode(dynamic data, {bool allowNull = true}) {
    if (data is DataRequestStatus) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'received':
          return DataRequestStatus.received;
        case r'in_progress':
          return DataRequestStatus.inProgress;
        case r'fulfilled':
          return DataRequestStatus.fulfilled;
        case r'refused':
          return DataRequestStatus.refused;
        case r'withdrawn':
          return DataRequestStatus.withdrawn;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static DataRequestStatusTypeTransformer? _instance;
}
