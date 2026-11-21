//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

enum SubscriptionInvoiceStatus {
  open._(r'open'),
  paid._(r'paid'),
  void_._(r'void'),
  ;

  /// Instantiate a new enum with the provided value.
  const SubscriptionInvoiceStatus._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [SubscriptionInvoiceStatus] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static SubscriptionInvoiceStatus? fromJson(dynamic value) =>
      SubscriptionInvoiceStatusTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [SubscriptionInvoiceStatus]
  /// that were successfully decoded from the passed [JSON][json].
  static List<SubscriptionInvoiceStatus> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SubscriptionInvoiceStatus>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SubscriptionInvoiceStatus.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [SubscriptionInvoiceStatus] to String,
/// and [decode] dynamic data back to [SubscriptionInvoiceStatus].
class SubscriptionInvoiceStatusTypeTransformer {
  factory SubscriptionInvoiceStatusTypeTransformer() =>
      _instance ??= const SubscriptionInvoiceStatusTypeTransformer._();

  const SubscriptionInvoiceStatusTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(SubscriptionInvoiceStatus data) => data._value;

  /// Returns the instance of [SubscriptionInvoiceStatus] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  SubscriptionInvoiceStatus? decode(dynamic data, {bool allowNull = true}) {
    if (data is SubscriptionInvoiceStatus) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'open':
          return SubscriptionInvoiceStatus.open;
        case r'paid':
          return SubscriptionInvoiceStatus.paid;
        case r'void':
          return SubscriptionInvoiceStatus.void_;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static SubscriptionInvoiceStatusTypeTransformer? _instance;
}
