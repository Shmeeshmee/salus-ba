//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

/// Its report to the tax authority: `pending` until the job fiscalizes it, `failed` when the job gave up (the report is retried), `fiscalized` once, for good.
enum FiscalStatus {
  pending._(r'pending'),
  fiscalized._(r'fiscalized'),
  failed._(r'failed'),
  ;

  /// Instantiate a new enum with the provided value.
  const FiscalStatus._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [FiscalStatus] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static FiscalStatus? fromJson(dynamic value) =>
      FiscalStatusTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [FiscalStatus]
  /// that were successfully decoded from the passed [JSON][json].
  static List<FiscalStatus> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <FiscalStatus>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = FiscalStatus.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [FiscalStatus] to String,
/// and [decode] dynamic data back to [FiscalStatus].
class FiscalStatusTypeTransformer {
  factory FiscalStatusTypeTransformer() =>
      _instance ??= const FiscalStatusTypeTransformer._();

  const FiscalStatusTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(FiscalStatus data) => data._value;

  /// Returns the instance of [FiscalStatus] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  FiscalStatus? decode(dynamic data, {bool allowNull = true}) {
    if (data is FiscalStatus) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'pending':
          return FiscalStatus.pending;
        case r'fiscalized':
          return FiscalStatus.fiscalized;
        case r'failed':
          return FiscalStatus.failed;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static FiscalStatusTypeTransformer? _instance;
}
